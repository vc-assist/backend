package server

import (
	"context"
	"time"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"

	"connectrpc.com/connect"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

type sessionCache struct {
	cache    *expirable.LRU[string, view.Client]
	keychain keychainv1connect.KeychainServiceClient
}

func newSessionCache(keychain keychainv1connect.KeychainServiceClient) sessionCache {
	return sessionCache{
		cache:    expirable.NewLRU[string, view.Client](2048, nil, time.Minute*15),
		keychain: keychain,
	}
}

func (s sessionCache) Get(ctx context.Context, email string) (view.Client, error) {
	cached, hit := s.cache.Get(email)
	if hit {
		return cached, nil
	}

	res, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        email,
		},
	})
	if err != nil {
		return view.Client{}, err
	}

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: baseUrl,
	})
	if err != nil {
		return view.Client{}, err
	}
	err = coreClient.LoginUsernamePassword(
		ctx,
		res.Msg.GetKey().GetUsername(),
		res.Msg.GetKey().GetPassword(),
	)
	if err != nil {
		return view.Client{}, err
	}
	client, err := view.NewClient(ctx, coreClient)
	if err != nil {
		return view.Client{}, err
	}

	s.cache.Add(email, client)
	return client, nil
}
