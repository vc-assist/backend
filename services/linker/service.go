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
)

var tracer = otel.Tracer("services/linker")

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

	var leftList []string
	for _, left := range req.Msg.GetSrc().GetKeys() {
		_, ok := mapping[left]
		if ok {
			continue
		}
		leftList = append(leftList, left)
	}
	var rightList []string
	for _, right := range req.Msg.GetDst().GetKeys() {
		_, ok := mapping[right]
		if ok {
			continue
		}
		rightList = append(rightList, right)
	}
	implicit := CreateImplicitLinks(leftList, rightList)
	for _, link := range implicit {
		if link.Correlation < 0.75 {
			continue
		}
		mapping[link.Left] = link.Right
	}

	return &connect.Response[linkerv1.LinkResponse]{
		Msg: &linkerv1.LinkResponse{
			SrcToDst: mapping,
		},
	}, nil
}

func (s Service) GetKnownSets(ctx context.Context, req *connect.Request[linkerv1.GetKnownSetsRequest]) (*connect.Response[linkerv1.GetKnownSetsResponse], error) {
	ctx, span := tracer.Start(ctx, "GetKnownSets")
	defer span.End()

	sets, err := s.qry.GetKnownSets(ctx)
	if err != nil {
		return nil, err
	}

	return &connect.Response[linkerv1.GetKnownSetsResponse]{
		Msg: &linkerv1.GetKnownSetsResponse{Sets: sets},
	}, nil
}

func (s Service) GetKnownKeys(ctx context.Context, req *connect.Request[linkerv1.GetKnownKeysRequest]) (*connect.Response[linkerv1.GetKnownKeysResponse], error) {
	ctx, span := tracer.Start(ctx, "GetKnownKeys")
	defer span.End()

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
