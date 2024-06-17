package powerschool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type graphqlQueryObject struct {
	Name      string `json:"operationName"`
	Variables any    `json:"variables"`
	Query     string `json:"query"`
}

type graphqlQueryResult[Data any] struct {
	Data Data `json:"data"`
}

func graphqlQuery[Input, Output any](
	ctx context.Context,
	client *resty.Client,
	name,
	query string,
	variables Input,
) (Output, error) {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("graphql:%s", name))
	defer span.End()

	obj := graphqlQueryObject{
		Name:      name,
		Query:     query,
		Variables: variables,
	}

	span.SetAttributes(attribute.KeyValue{
		Key:   "custom.name",
		Value: attribute.StringValue(name),
	})
	serialized, err := json.Marshal(variables)
	if err == nil {
		span.SetAttributes(attribute.KeyValue{
			Key:   "custom.variables",
			Value: attribute.StringValue(string(serialized)),
		})
	} else {
		span.SetAttributes(attribute.KeyValue{
			Key:   "custom.variables",
			Value: attribute.StringValue("ERROR: failed to serialize variables."),
		})
	}

	var defaultOut Output

	body, err := json.Marshal(obj)
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

	var result graphqlQueryResult[Output]
	err = json.Unmarshal(res.Body(), &result)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse json response")
		return defaultOut, err
	}

	return result.Data, nil
}
