package view

import (
	"bytes"
	"context"
	"encoding/gob"
	"net/url"
	"vcassist-backend/lib/htmlutil"
	"vcassist-backend/lib/timezone"

	"github.com/PuerkitoBio/purell"
	"github.com/dgraph-io/badger/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var errWebpageNotFound = badger.ErrKeyNotFound

type webpage struct {
	Contents []byte
	Anchors  []htmlutil.Anchor

	ExpiresAt int64
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
			purell.FlagRemoveDirectoryIndex|
			purell.FlagRemoveFragment|
			purell.FlagSortQuery,
	)
	key := clientId + ":" + normalized
	return key, nil
}

func (c webpageCache) get(ctx context.Context, clientId, endpoint string) (webpage, error) {
	ctx, span := tracer.Start(ctx, "get")
	defer span.End()

	key, err := c.key(clientId, endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create cache key")
		return webpage{}, err
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "cache_key",
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

	decoder := gob.NewDecoder(bytes.NewBuffer(serialized))

	var cached webpage
	err = decoder.Decode(&cached)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deserialize cached item")
		return webpage{}, err
	}

	if timezone.Now().Unix() >= cached.ExpiresAt {
		span.AddEvent("delete expired cache key", trace.WithAttributes(attribute.KeyValue{
			Key:   "key",
			Value: attribute.StringValue(key),
		}))

		tx := c.db.NewTransaction(true)
		defer tx.Commit()

		err = tx.Delete([]byte(key))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to delete expired key")
			return webpage{}, errWebpageNotFound
		}

		span.SetStatus(codes.Ok, "CACHE EXPIRED")
		return webpage{}, errWebpageNotFound
	}

	span.AddEvent(
		"successfully returned cached webpage",
		trace.WithAttributes(
			attribute.KeyValue{
				Key:   "contentlength",
				Value: attribute.IntValue(len(cached.Contents)),
			},
			attribute.KeyValue{
				Key:   "anchorlength",
				Value: attribute.IntValue(len(cached.Anchors)),
			},
		),
	)

	return cached, nil
}

func (c webpageCache) set(ctx context.Context, clientId, endpoint string, page webpage) error {
	ctx, span := tracer.Start(ctx, "set")
	defer span.End()

	key, err := c.key(clientId, endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create cache key")
		return err
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "cache_key",
		Value: attribute.StringValue(key),
	})

	serialized := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(serialized)
	err = encoder.Encode(page)
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
