package powerschool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func serializeGraphqlQueryObject(name, query string, variables protoreflect.ProtoMessage) (string, error) {
	escapedName, err := json.Marshal(name)
	if err != nil {
		return "", err
	}
	escapedQuery, err := json.Marshal(query)
	if err != nil {
		return "", err
	}
	jsonVariables, err := protojson.Marshal(variables)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		`{"operationName": %s, "query": %s, "variables": %s}`,
		escapedName,
		escapedQuery,
		jsonVariables,
	), nil
}

func deserializeGraphqlResponseObject(response []byte, out protoreflect.ProtoMessage) error {
	var result struct {
		Data any `json:"data"`
	}

	err := json.Unmarshal(response, &result)
	if err != nil {
		return err
	}

	// this is extremely inefficient, please fix this later
	dataResult, err := json.Marshal(result.Data)
	if err != nil {
		return err
	}

	return protojson.Unmarshal(dataResult, out)
}

func graphqlQuery(
	ctx context.Context,
	client *resty.Client,
	name,
	query string,
	variables protoreflect.ProtoMessage,
	output protoreflect.ProtoMessage,
) error {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("graphql:%s", name))
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "name",
		Value: attribute.StringValue(name),
	})
	serialized, err := json.Marshal(variables)
	if err == nil {
		span.SetAttributes(attribute.KeyValue{
			Key:   "variables",
			Value: attribute.StringValue(string(serialized)),
		})
	} else {
		span.SetAttributes(attribute.KeyValue{
			Key:   "variables",
			Value: attribute.StringValue("ERROR: failed to serialize variables."),
		})
	}

	body, err := serializeGraphqlQueryObject(name, query, variables)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize json query")
		return err
	}

	res, err := client.R().
		SetContext(ctx).
		SetHeader("content-type", "application/json").
		SetBody(body).
		Post("https://mobile.powerschool.com/v3.0/graphql")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return err
	}

	err = deserializeGraphqlResponseObject(res.Body(), output)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse json response")
		return err
	}

	return nil
}
