package moodle_student

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

var db *badger.DB
var baseUrl *url.URL

func init() {
	var err error
	baseUrl, err = url.Parse("https://learn.vcs.net")
	if err != nil {
		panic(err)
	}
	db, err = badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}
}

func TestCacheKey(t *testing.T) {
	cache := webpageCache{baseUrl: baseUrl}

	testCases := []struct {
		clientId string
		endpoint string
		expect   string
	}{
		{clientId: "clientA", endpoint: "/index.php", expect: "clientA:https://learn.vcs.net/"},
		{clientId: "client b", endpoint: "https://google.com", expect: "client b:https://google.com/"},
		{clientId: "client b", endpoint: "https://google.com/index.html", expect: "client b:https://google.com/"},
		{clientId: "client b", endpoint: "https://www.google.com?b=2&a=1#1-2", expect: "client b:https://www.google.com/?a=1&b=2"},
		{clientId: "clientA", endpoint: "/", expect: "clientA:https://learn.vcs.net/"},
	}

	for _, test := range testCases {
		res, err := cache.key(test.clientId, test.endpoint)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, res, test.expect)
	}
}

func TestCache(t *testing.T) {
	cache := webpageCache{
		baseUrl: baseUrl,
		db:      db,
	}

	_, err := cache.get(context.Background(), "client_a", "/index.php")
	require.Equal(t, err, errWebpageNotFound)

	page1original := webpage{
		Anchors: []Chapter{
			{
				Name: "book 1",
				Href: "https://learn.vcs.net/view.php?chapterid=0",
			},
			{
				Name: "book 2",
				Href: "https://learn.vcs.net/view.php?chapterid=1",
			},
		},
		Contents:  []byte("some webpage contents"),
		ExpiresAt: int64(time.Duration(time.Now().Unix()) + 1),
	}
	err = cache.set(context.Background(), "client_a", "/mod/book/view.php?id=1", page1original)
	require.Nil(t, err)

	_, err = cache.get(context.Background(), "client_b", "/mod/book/view.php?id=1")
	require.Equal(t, err, errWebpageNotFound)

	page1cached, err := cache.get(context.Background(), "client_a", "/mod/book/view.php?id=1")
	require.Nil(t, err)
	diff := cmp.Diff(page1original, page1cached)
	require.Empty(t, diff)

	time.Sleep(time.Second)
	_, err = cache.get(context.Background(), "client_a", "/mod/book/view.php?id=1")
	require.Equal(t, err, errWebpageNotFound)
}
