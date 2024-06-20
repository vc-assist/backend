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

func deserializeGraphqlResponseObject[T protoreflect.ProtoMessage](response []byte) (T, error) {
	var out T
	var result struct {
		Data any `json:"data"`
	}

	err := json.Unmarshal(response, &result)
	if err != nil {
		return out, err
	}

	// this is extremely inefficient, please fix this later
	dataResult, err := json.Marshal(result.Data)
	if err != nil {
		return out, err
	}

	err = protojson.Unmarshal(dataResult, out)
	return out, err
}

func graphqlQuery[Input, Output protoreflect.ProtoMessage](
	ctx context.Context,
	client *resty.Client,
	name,
	query string,
	variables Input,
) (Output, error) {
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

	var defaultOut Output

	body, err := serializeGraphqlQueryObject(name, query, variables)
	if err != nil {
		span.SetStatus(codes.Error, "failed to serialize json query")
		return defaultOut, err
	}

	res, err := client.R().
		SetContext(ctx).
		SetHeader("content-type", "application/json").
		SetBody(body).
		Post("/v3.0/graphql")
	if err != nil {
		span.SetStatus(codes.Error, "failed to fetch")
		return defaultOut, err
	}

	result, err := deserializeGraphqlResponseObject[Output](res.Body())
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse json response")
		return defaultOut, err
	}

	return result, nil
}
