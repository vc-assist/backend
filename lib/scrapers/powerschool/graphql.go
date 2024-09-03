package powerschool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type graphqlRequest struct {
	Name     string `json:"operationName"`
	Query    string `json:"query"`
	Variable any    `json:"variables"`
}

type graphqlResponse[T any] struct {
	Data T `json:"data"`
}

func graphqlQuery[O any](
	ctx context.Context,
	client *resty.Client,
	name,
	query string,
	variables any,
	output *O,
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

	body, err := json.Marshal(graphqlRequest{
		Name:     name,
		Query:    query,
		Variable: variables,
	})
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

	parsed := graphqlResponse[O]{}
	err = json.Unmarshal(res.Body(), &parsed)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse json response")
		return err
	}

	*output = parsed.Data

	if span.IsRecording() {
		debugInfo, _ := json.Marshal(output)
		span.SetAttributes(attribute.String("output", string(debugInfo)))
	}

	return nil
}
