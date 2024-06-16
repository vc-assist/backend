package moodlestudent

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/url"
	"vcassist-backend/lib/htmlutil"

	"github.com/PuerkitoBio/purell"
	"github.com/dgraph-io/badger/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var errWebpageNotFound = badger.ErrKeyNotFound

type webpage struct {
	contents []byte
	anchors  []htmlutil.Anchor

	createdAt int64
	lifetime  int64
}

type webpageCache struct {
	db      *badger.DB
	baseUrl *url.URL
}

func (c webpageCache) key(clientId, endpoint string) (string, error) {
	full, err := c.baseUrl.Parse(endpoint)
	if err != nil {
		return "", err
	}
	normalized := purell.NormalizeURL(
		full,
		purell.FlagsSafe|
			purell.FlagsUsuallySafeNonGreedy|
			purell.FlagsUnsafeNonGreedy,
	)
	key := clientId + ":" + normalized
	return key, nil
}

func (c webpageCache) get(ctx context.Context, clientId, endpoint string) (webpage, error) {
	ctx, span := tracer.Start(ctx, "cache:get")
	defer span.End()

	key, err := c.key(clientId, endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create cache key")
		return webpage{}, err
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "custom.cache_key",
		Value: attribute.StringValue(key),
	})

	tx := c.db.NewTransaction(false)
	defer tx.Discard()
	item, err := tx.Get([]byte(key))
	if err == badger.ErrKeyNotFound {
		return webpage{}, errWebpageNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read item from badger")
		return webpage{}, err
	}
	serialized, err := item.ValueCopy(nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to copy cached item")
		return webpage{}, err
	}

	var cached webpage
	err = binary.Read(bytes.NewBuffer(serialized), binary.BigEndian, &cached)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deserialize cached item")
		return webpage{}, err
	}

	span.AddEvent(
		"Successfully returned cached webpage.",
		trace.WithAttributes(
			attribute.KeyValue{
				Key:   "custom.contentlength",
				Value: attribute.IntValue(len(cached.contents)),
			},
			attribute.KeyValue{
				Key:   "custom.anchorlength",
				Value: attribute.IntValue(len(cached.anchors)),
			},
		),
	)

	return cached, nil
}

func (c webpageCache) set(ctx context.Context, clientId, endpoint string, page webpage) error {
	ctx, span := tracer.Start(ctx, "cache:set")
	defer span.End()

	key, err := c.key(clientId, endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create cache key")
		return err
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "custom.cache_key",
		Value: attribute.StringValue(key),
	})

	serialized := bytes.NewBuffer(nil)
	err = binary.Write(serialized, binary.BigEndian, page)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize webpage")
		return err
	}

	tx := c.db.NewTransaction(true)
	defer tx.Commit()

	err = tx.Set([]byte(key), serialized.Bytes())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to set badger item")
		return err
	}

	return nil
}
