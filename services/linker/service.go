package linker

import (
	"context"
	"database/sql"
	"vcassist-backend/lib/timezone"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var tracer = otel.Tracer("vcassist.services.linker")

type Service struct {
	qry *db.Queries
	db  *sql.DB
}

func NewService(database *sql.DB) linkerv1connect.LinkerServiceClient {
	return linkerv1connect.NewInstrumentedLinkerServiceClient(
		Service{
			qry: db.New(database),
			db:  database,
		},
	)
}

func (s Service) GetExplicitLinks(ctx context.Context, req *connect.Request[linkerv1.GetExplicitLinksRequest]) (*connect.Response[linkerv1.GetExplicitLinksResponse], error) {
	links, err := s.qry.GetExplicitLinks(ctx, db.GetExplicitLinksParams{
		Leftset:  req.Msg.GetLeftSet(),
		Rightset: req.Msg.GetRightSet(),
	})
	if err != nil {
		return nil, err
	}

	var leftKeys []string
	var rightKeys []string
	for _, l := range links {
		if l.Rightset == req.Msg.GetLeftSet() {
			leftKeys = append(leftKeys, l.Rightkey)
			rightKeys = append(rightKeys, l.Leftkey)
			continue
		}
		leftKeys = append(leftKeys, l.Leftkey)
		rightKeys = append(rightKeys, l.Rightkey)
	}

	return &connect.Response[linkerv1.GetExplicitLinksResponse]{
		Msg: &linkerv1.GetExplicitLinksResponse{
			LeftKeys:  leftKeys,
			RightKeys: rightKeys,
		},
	}, nil
}

func (s Service) AddExplicitLink(ctx context.Context, req *connect.Request[linkerv1.AddExplicitLinkRequest]) (*connect.Response[linkerv1.AddExplicitLinkResponse], error) {
	err := s.qry.CreateExplicitLink(ctx, db.CreateExplicitLinkParams{
		Leftset:  req.Msg.GetLeft().GetSet(),
		Leftkey:  req.Msg.GetLeft().GetKey(),
		Rightset: req.Msg.GetRight().GetSet(),
		Rightkey: req.Msg.GetRight().GetKey(),
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[linkerv1.AddExplicitLinkResponse]{Msg: &linkerv1.AddExplicitLinkResponse{}}, nil
}

func (s Service) DeleteExplicitLink(ctx context.Context, req *connect.Request[linkerv1.DeleteExplicitLinkRequest]) (*connect.Response[linkerv1.DeleteExplicitLinkResponse], error) {
	err := s.qry.DeleteExplicitLink(ctx, db.DeleteExplicitLinkParams{
		Leftset:  req.Msg.GetLeft().GetSet(),
		Leftkey:  req.Msg.GetLeft().GetKey(),
		Rightset: req.Msg.GetRight().GetSet(),
		Rightkey: req.Msg.GetRight().GetKey(),
	})
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txqry := s.qry.WithTx(tx)
	err = txqry.CreateKnownSet(ctx, req.Msg.GetSrc().GetName())
	if err != nil {
		return nil, err
	}
	err = txqry.CreateKnownSet(ctx, req.Msg.GetDst().GetName())
	if err != nil {
		return nil, err
	}

	now := timezone.Now().Unix()
	for _, src := range req.Msg.GetSrc().GetKeys() {
		err = txqry.CreateKnownKey(ctx, db.CreateKnownKeyParams{
			Setname:  req.Msg.GetSrc().GetName(),
			Value:    src,
			Lastseen: now,
		})
		if err != nil {
			return nil, err
		}
	}
	for _, dst := range req.Msg.GetDst().GetKeys() {
		err = txqry.CreateKnownKey(ctx, db.CreateKnownKeyParams{
			Setname:  req.Msg.GetDst().GetName(),
			Value:    dst,
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

	explicit, err := s.GetExplicitLinks(ctx, &connect.Request[linkerv1.GetExplicitLinksRequest]{
		Msg: &linkerv1.GetExplicitLinksRequest{
			LeftSet:  req.Msg.GetSrc().GetName(),
			RightSet: req.Msg.GetDst().GetName(),
		},
	})
	if err != nil {
		return nil, err
	}

	mapping := make(map[string]string)
	for i, left := range explicit.Msg.GetLeftKeys() {
		right := explicit.Msg.GetRightKeys()[i]
		mapping[left] = right
	}

	var exactMatches []string
	if len(req.Msg.GetSrc().GetKeys()) <= len(req.Msg.GetDst().GetKeys()) {
		dstKeys := make(map[string]struct{})
		for _, dst := range req.Msg.GetDst().GetKeys() {
			dstKeys[dst] = struct{}{}
		}
		for _, src := range req.Msg.GetSrc().GetKeys() {
			_, hasKey := dstKeys[src]
			if hasKey {
				exactMatches = append(exactMatches, src)
			}
		}
	} else {
		srcKeys := make(map[string]struct{})
		for _, src := range req.Msg.GetSrc().GetKeys() {
			srcKeys[src] = struct{}{}
		}
		for _, dst := range req.Msg.GetDst().GetKeys() {
			_, hasKey := srcKeys[dst]
			if hasKey {
				exactMatches = append(exactMatches, dst)
			}
		}
	}
	for _, k := range exactMatches {
		mapping[k] = k
	}

	return &connect.Response[linkerv1.LinkResponse]{
		Msg: &linkerv1.LinkResponse{
			SrcToDst: mapping,
		},
	}, nil
}

func (s Service) SuggestLinks(ctx context.Context, req *connect.Request[linkerv1.SuggestLinksRequest]) (*connect.Response[linkerv1.SuggestLinksResponse], error) {
	leftRes, err := s.GetKnownKeys(ctx, &connect.Request[linkerv1.GetKnownKeysRequest]{
		Msg: &linkerv1.GetKnownKeysRequest{
			Set: req.Msg.GetSetLeft(),
		},
	})
	if err != nil {
		return nil, err
	}
	rightRes, err := s.GetKnownKeys(ctx, &connect.Request[linkerv1.GetKnownKeysRequest]{
		Msg: &linkerv1.GetKnownKeysRequest{
			Set: req.Msg.GetSetRight(),
		},
	})
	if err != nil {
		return nil, err
	}

	leftKeys := make([]string, len(leftRes.Msg.GetKeys()))
	for i := 0; i < len(leftRes.Msg.GetKeys()); i++ {
		leftKeys[i] = leftRes.Msg.GetKeys()[i].GetKey()
	}
	rightKeys := make([]string, len(rightRes.Msg.GetKeys()))
	for i := 0; i < len(rightRes.Msg.GetKeys()); i++ {
		rightKeys[i] = rightRes.Msg.GetKeys()[i].GetKey()
	}

	implicit := CreateImplicitLinks(leftKeys, rightKeys)

	suggestions := []*linkerv1.LinkSuggestion{}
	for _, impl := range implicit {
		if impl.Correlation < 0.75 || impl.Correlation == 1 {
			continue
		}
		suggestions = append(suggestions, &linkerv1.LinkSuggestion{
			LeftKey:  impl.Left,
			RightKey: impl.Right,
		})
	}

	return &connect.Response[linkerv1.SuggestLinksResponse]{
		Msg: &linkerv1.SuggestLinksResponse{
			Suggestions: suggestions,
		},
	}, nil
}
