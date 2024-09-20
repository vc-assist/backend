package linker

import (
	"context"
	"database/sql"
	"log/slog"
	"vcassist-backend/lib/timezone"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"
	"vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	_ "modernc.org/sqlite"
)

type Service struct {
	qry *db.Queries
	db  *sql.DB
}

func NewService(database *sql.DB) Service {
	return Service{
		qry: db.New(database),
		db:  database,
	}
}

func (s Service) GetExplicitLinks(ctx context.Context, req *connect.Request[linkerv1.GetExplicitLinksRequest]) (*connect.Response[linkerv1.GetExplicitLinksResponse], error) {
	left := req.Msg.GetLeftSet()
	right := req.Msg.GetRightSet()

	links, err := s.qry.GetExplicitLinks(ctx, db.GetExplicitLinksParams{
		Leftset:  left,
		Rightset: right,
	})
	if err != nil {
		return nil, err
	}

	leftKeys := make([]string, len(links))
	rightKeys := make([]string, len(links))
	for i, l := range links {
		leftKeys[i] = l.Leftkey
		rightKeys[i] = l.Rightkey
	}

	return &connect.Response[linkerv1.GetExplicitLinksResponse]{
		Msg: &linkerv1.GetExplicitLinksResponse{
			LeftKeys:  leftKeys,
			RightKeys: rightKeys,
		},
	}, nil
}

func (s Service) AddExplicitLink(ctx context.Context, req *connect.Request[linkerv1.AddExplicitLinkRequest]) (*connect.Response[linkerv1.AddExplicitLinkResponse], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	err = txqry.CreateExplicitLink(ctx, db.CreateExplicitLinkParams{
		Leftset:  req.Msg.GetLeft().GetSet(),
		Leftkey:  req.Msg.GetLeft().GetKey(),
		Rightset: req.Msg.GetRight().GetSet(),
		Rightkey: req.Msg.GetRight().GetKey(),
	})
	if err != nil {
		return nil, err
	}
	err = txqry.CreateExplicitLink(ctx, db.CreateExplicitLinkParams{
		Rightset: req.Msg.GetLeft().GetSet(),
		Rightkey: req.Msg.GetLeft().GetKey(),
		Leftset:  req.Msg.GetRight().GetSet(),
		Leftkey:  req.Msg.GetRight().GetKey(),
	})
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &connect.Response[linkerv1.AddExplicitLinkResponse]{Msg: &linkerv1.AddExplicitLinkResponse{}}, nil
}

func (s Service) DeleteExplicitLink(ctx context.Context, req *connect.Request[linkerv1.DeleteExplicitLinkRequest]) (*connect.Response[linkerv1.DeleteExplicitLinkResponse], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	err = txqry.DeleteExplicitLink(ctx, db.DeleteExplicitLinkParams{
		Leftset:  req.Msg.GetLeft().GetSet(),
		Leftkey:  req.Msg.GetLeft().GetKey(),
		Rightset: req.Msg.GetRight().GetSet(),
		Rightkey: req.Msg.GetRight().GetKey(),
	})
	if err != nil {
		return nil, err
	}
	err = txqry.DeleteExplicitLink(ctx, db.DeleteExplicitLinkParams{
		Leftset:  req.Msg.GetRight().GetSet(),
		Leftkey:  req.Msg.GetRight().GetKey(),
		Rightset: req.Msg.GetLeft().GetSet(),
		Rightkey: req.Msg.GetLeft().GetKey(),
	})
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &connect.Response[linkerv1.DeleteExplicitLinkResponse]{
		Msg: &linkerv1.DeleteExplicitLinkResponse{},
	}, nil
}

func (s Service) GetKnownSets(ctx context.Context, req *connect.Request[linkerv1.GetKnownSetsRequest]) (*connect.Response[linkerv1.GetKnownSetsResponse], error) {
	sets, err := s.qry.GetKnownSets(ctx)
	if err != nil {
		return nil, err
	}

	return &connect.Response[linkerv1.GetKnownSetsResponse]{
		Msg: &linkerv1.GetKnownSetsResponse{Sets: sets},
	}, nil
}

func (s Service) GetKnownKeys(ctx context.Context, req *connect.Request[linkerv1.GetKnownKeysRequest]) (*connect.Response[linkerv1.GetKnownKeysResponse], error) {
	rows, err := s.qry.GetKnownKeys(ctx, req.Msg.GetSet())
	if err != nil {
		return nil, err
	}

	keys := make([]*linkerv1.KnownKey, len(rows))
	for i, r := range rows {
		keys[i] = &linkerv1.KnownKey{
			Key:      r.Value,
			LastSeen: r.Lastseen,
		}
	}

	return &connect.Response[linkerv1.GetKnownKeysResponse]{
		Msg: &linkerv1.GetKnownKeysResponse{
			Keys: keys,
		},
	}, nil
}

func (s Service) DeleteKnownSets(ctx context.Context, req *connect.Request[linkerv1.DeleteKnownSetsRequest]) (*connect.Response[linkerv1.DeleteKnownSetsResponse], error) {
	err := s.qry.DeleteKnownSets(ctx, req.Msg.GetSets())
	if err != nil {
		return nil, err
	}
	return &connect.Response[linkerv1.DeleteKnownSetsResponse]{
		Msg: &linkerv1.DeleteKnownSetsResponse{},
	}, nil
}

func (s Service) DeleteKnownKeys(ctx context.Context, req *connect.Request[linkerv1.DeleteKnownKeysRequest]) (*connect.Response[linkerv1.DeleteKnownKeysResponse], error) {
	err := s.qry.DeleteKeysBefore(ctx, db.DeleteKeysBeforeParams{
		Setname:  req.Msg.GetSet(),
		Lastseen: req.Msg.GetBefore(),
	})
	if err != nil {
		return nil, err
	}
	return &connect.Response[linkerv1.DeleteKnownKeysResponse]{
		Msg: &linkerv1.DeleteKnownKeysResponse{},
	}, nil
}

func (s Service) Link(ctx context.Context, req *connect.Request[linkerv1.LinkRequest]) (*connect.Response[linkerv1.LinkResponse], error) {
	left := req.Msg.GetSrc().GetName()
	leftKeys := req.Msg.GetSrc().GetKeys()
	right := req.Msg.GetDst().GetName()
	rightKeys := req.Msg.GetDst().GetKeys()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txqry := s.qry.WithTx(tx)

	slog.DebugContext(ctx, "create known sets")

	err = txqry.CreateKnownSet(ctx, left)
	if err != nil {
		return nil, err
	}
	err = txqry.CreateKnownSet(ctx, right)
	if err != nil {
		return nil, err
	}

	now := timezone.Now().Unix()
	for _, key := range leftKeys {
		err = txqry.CreateKnownKey(ctx, db.CreateKnownKeyParams{
			Setname:  left,
			Value:    key,
			Lastseen: now,
		})
		if err != nil {
			return nil, err
		}
	}
	for _, key := range rightKeys {
		err = txqry.CreateKnownKey(ctx, db.CreateKnownKeyParams{
			Setname:  right,
			Value:    key,
			Lastseen: now,
		})
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	links, err := s.qry.GetExplicitLinks(ctx, db.GetExplicitLinksParams{
		Leftset:  left,
		Rightset: right,
	})
	if err != nil {
		return nil, err
	}

	slog.DebugContext(ctx, "explicit links", "links", links)

	mapping := make(map[string]string)

	srcKeys := make(map[string]struct{})
	for _, k := range leftKeys {
		srcKeys[k] = struct{}{}
	}
	slog.DebugContext(ctx, "source keys", "keys", srcKeys)

	for _, k := range rightKeys {
		_, ok := srcKeys[k]
		if !ok {
			continue
		}
		slog.DebugContext(ctx, "exact match", "key", k)
		mapping[k] = k
	}
	for _, l := range links {
		_, ok := srcKeys[l.Leftkey]
		if !ok {
			continue
		}
		mapping[l.Leftkey] = l.Rightkey
	}

	return &connect.Response[linkerv1.LinkResponse]{
		Msg: &linkerv1.LinkResponse{
			SrcToDst: mapping,
		},
	}, nil
}

func (s Service) SuggestLinks(ctx context.Context, req *connect.Request[linkerv1.SuggestLinksRequest]) (*connect.Response[linkerv1.SuggestLinksResponse], error) {
	span := trace.SpanFromContext(ctx)

	left := req.Msg.GetSetLeft()
	right := req.Msg.GetSetRight()

	leftRes, err := s.GetKnownKeys(ctx, &connect.Request[linkerv1.GetKnownKeysRequest]{
		Msg: &linkerv1.GetKnownKeysRequest{
			Set: left,
		},
	})
	if err != nil {
		return nil, err
	}
	rightRes, err := s.GetKnownKeys(ctx, &connect.Request[linkerv1.GetKnownKeysRequest]{
		Msg: &linkerv1.GetKnownKeysRequest{
			Set: right,
		},
	})
	if err != nil {
		return nil, err
	}

	explicit, err := s.GetExplicitLinks(ctx, &connect.Request[linkerv1.GetExplicitLinksRequest]{
		Msg: &linkerv1.GetExplicitLinksRequest{
			LeftSet:  left,
			RightSet: right,
		},
	})
	if err != nil {
		return nil, err
	}
	resolvedLeftKeys := make(map[string]struct{})
	for _, k := range explicit.Msg.GetLeftKeys() {
		resolvedLeftKeys[k] = struct{}{}
	}
	resolvedRightKeys := make(map[string]struct{})
	for _, k := range explicit.Msg.GetRightKeys() {
		resolvedRightKeys[k] = struct{}{}
	}

	leftKeys := []string{}
	for i := 0; i < len(leftRes.Msg.GetKeys()); i++ {
		key := leftRes.Msg.GetKeys()[i].GetKey()
		_, resolved := resolvedLeftKeys[key]
		if resolved {
			continue
		}
		leftKeys = append(leftKeys, key)
	}
	rightKeys := []string{}
	for i := 0; i < len(rightRes.Msg.GetKeys()); i++ {
		key := rightRes.Msg.GetKeys()[i].GetKey()
		_, resolved := resolvedRightKeys[key]
		if resolved {
			continue
		}
		rightKeys = append(rightKeys, key)
	}

	span.AddEvent("left keys", trace.WithAttributes(
		attribute.StringSlice("keys", leftKeys),
	))
	span.AddEvent("right keys", trace.WithAttributes(
		attribute.StringSlice("keys", rightKeys),
	))

	implicit := CreateImplicitLinks(leftKeys, rightKeys)
	threshold := float64(req.Msg.GetThreshold())

	suggestions := []*linkerv1.LinkSuggestion{}
	for _, impl := range implicit {
		if impl.Correlation < threshold || impl.Correlation == 1 {
			continue
		}
		suggestions = append(suggestions, &linkerv1.LinkSuggestion{
			LeftKey:     impl.Left,
			RightKey:    impl.Right,
			Correlation: float32(impl.Correlation),
		})
	}

	return &connect.Response[linkerv1.SuggestLinksResponse]{
		Msg: &linkerv1.SuggestLinksResponse{
			Suggestions: suggestions,
		},
	}, nil
}
