package linker

import (
	"context"
	"database/sql"
	"vcassist-backend/services/linker/api"
	"vcassist-backend/services/linker/db"
	"vcassist-backend/lib/timezone"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("services/linker")

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

func (s Service) GetExplicitLinks(ctx context.Context, req *connect.Request[api.GetExplicitLinksRequest]) (*connect.Response[api.GetExplicitLinksResponse], error) {
	ctx, span := tracer.Start(ctx, "GetExplicitLinks")
	defer span.End()

	links, err := s.qry.GetExplicitLinks(ctx, db.GetExplicitLinksParams{
		Leftset:  req.Msg.LeftSet,
		Rightset: req.Msg.RightSet,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var leftKeys []string
	var rightKeys []string
	for _, l := range links {
		if l.Rightset == req.Msg.LeftSet {
			leftKeys = append(leftKeys, l.Rightkey)
			rightKeys = append(rightKeys, l.Leftkey)
			continue
		}
		leftKeys = append(leftKeys, l.Leftkey)
		rightKeys = append(rightKeys, l.Rightkey)
	}

	return &connect.Response[api.GetExplicitLinksResponse]{
		Msg: &api.GetExplicitLinksResponse{
			LeftKeys:  leftKeys,
			RightKeys: rightKeys,
		},
	}, nil
}

func (s Service) AddExplicitLink(ctx context.Context, req *connect.Request[api.AddExplicitLinkRequest]) (*connect.Response[api.AddExplicitLinkResponse], error) {
	ctx, span := tracer.Start(ctx, "AddExplicitLink")
	defer span.End()

	err := s.qry.CreateExplicitLink(ctx, db.CreateExplicitLinkParams{
		Leftset:  req.Msg.Left.Set,
		Leftkey:  req.Msg.Left.Key,
		Rightset: req.Msg.Right.Set,
		Rightkey: req.Msg.Right.Key,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.AddExplicitLinkResponse]{Msg: &api.AddExplicitLinkResponse{}}, nil
}

func (s Service) DeleteExplicitLink(ctx context.Context, req *connect.Request[api.DeleteExplicitLinkRequest]) (*connect.Response[api.DeleteExplicitLinkResponse], error) {
	ctx, span := tracer.Start(ctx, "DeleteExplicitLink")
	defer span.End()

	err := s.qry.DeleteExplicitLink(ctx, db.DeleteExplicitLinkParams{
		Leftset:  req.Msg.GetLeft().Set,
		Leftkey:  req.Msg.GetLeft().Key,
		Rightset: req.Msg.GetRight().Set,
		Rightkey: req.Msg.GetRight().Key,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.DeleteExplicitLinkResponse]{
		Msg: &api.DeleteExplicitLinkResponse{},
	}, nil
}

func (s Service) Link(ctx context.Context, req *connect.Request[api.LinkRequest]) (*connect.Response[api.LinkResponse], error) {
	ctx, span := tracer.Start(ctx, "Link")
	defer span.End()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer tx.Rollback()

	txqry := s.qry.WithTx(tx)
	err = txqry.CreateKnownSet(ctx, req.Msg.Src.Name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	err = txqry.CreateKnownSet(ctx, req.Msg.Dst.Name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	now := timezone.Now().Unix()
	for _, src := range req.Msg.Src.Keys {
		err = txqry.CreateKnownKey(ctx, db.CreateKnownKeyParams{
			Setname:  req.Msg.Src.Name,
			Value:    src,
			Lastseen: now,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	for _, dst := range req.Msg.Dst.Keys {
		err = txqry.CreateKnownKey(ctx, db.CreateKnownKeyParams{
			Setname:  req.Msg.Dst.Name,
			Value:    dst,
			Lastseen: now,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	explicit, err := s.GetExplicitLinks(ctx, &connect.Request[api.GetExplicitLinksRequest]{
		Msg: &api.GetExplicitLinksRequest{
			LeftSet:  req.Msg.Src.Name,
			RightSet: req.Msg.Dst.Name,
		},
	})
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]string)
	for i, left := range explicit.Msg.LeftKeys {
		right := explicit.Msg.RightKeys[i]
		mapping[left] = right
	}

	var leftList []string
	for _, left := range req.Msg.Src.Keys {
		_, ok := mapping[left]
		if ok {
			continue
		}
		leftList = append(leftList, left)
	}
	var rightList []string
	for _, right := range req.Msg.Dst.Keys {
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

	return &connect.Response[api.LinkResponse]{
		Msg: &api.LinkResponse{
			SrcToDst: mapping,
		},
	}, nil
}

func (s Service) GetKnownSets(ctx context.Context, req *connect.Request[api.GetKnownSetsRequest]) (*connect.Response[api.GetKnownSetsResponse], error) {
	ctx, span := tracer.Start(ctx, "GetKnownSets")
	defer span.End()

	sets, err := s.qry.GetKnownSets(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.GetKnownSetsResponse]{
		Msg: &api.GetKnownSetsResponse{Sets: sets},
	}, nil
}

func (s Service) GetKnownKeys(ctx context.Context, req *connect.Request[api.GetKnownKeysRequest]) (*connect.Response[api.GetKnownKeysResponse], error) {
	ctx, span := tracer.Start(ctx, "GetKnownKeys")
	defer span.End()

	rows, err := s.qry.GetKnownKeys(ctx, req.Msg.GetSet())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	keys := make([]*api.KnownKey, len(rows))
	for i, r := range rows {
		keys[i] = &api.KnownKey{
			Key:      r.Value,
			LastSeen: r.Lastseen,
		}
	}

	return &connect.Response[api.GetKnownKeysResponse]{
		Msg: &api.GetKnownKeysResponse{
			Keys: keys,
		},
	}, nil
}
