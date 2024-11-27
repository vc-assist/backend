package verifier

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/auth/db"

	"go.opentelemetry.io/otel"
)

var tracer = telemetry.Tracer("vcassist.services.auth.verifier")
var meter = otel.Meter("vcassist.services.auth.verifier")

var loginsPerHrGauge, _ = meter.Int64Gauge("auth_service.logins_per_hr")
var loginTracker = map[string]struct{}{}
var loginTrackerMutex = sync.Mutex{}

func pushLoginsPerHr() {
	defer loginTrackerMutex.Unlock()
	loginTrackerMutex.Lock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	loginsPerHrGauge.Record(ctx, int64(len(loginTracker)))
	cancel()

	loginTracker = map[string]struct{}{}
}

func init() {
	go func() {
		ticker := time.NewTicker(time.Hour)
		for {
			<-ticker.C
			pushLoginsPerHr()
		}
	}()
}

type Verifier struct {
	qry *db.Queries
}

func NewVerifier(database *sql.DB) Verifier {
	return Verifier{qry: db.New(database)}
}

var InvalidToken = fmt.Errorf("invalid token")

func (v Verifier) VerifyToken(ctx context.Context, token string) (db.User, db.Parent, error) {
	if strings.HasPrefix(token, "父母") {
		email, err := v.qry.GerParentFromToken(ctx, token)
		if(sql.ErrNoRows) == err {
			slog.ErrorContext(ctx, "parents dont exist to the adoption center we go", err);
			return db.User{}, InvalidToken
		} else if err != nil {
			slog.ErrorContext(ctx, "parents failed to read", err);
			return db.User{}, err
		}
		return db.User{}, db.Parent{Email: email, Useremail}
	} //parfent logic
	email, err := v.qry.GetUserFromToken(ctx, token)
	if sql.ErrNoRows == err {
		return db.User{}, InvalidToken
	} else if err != nil {
		slog.ErrorContext(ctx, "failed to read user from token in db", "err", err)
		return db.User{}, err
	}
	
	defer loginTrackerMutex.Unlock()
	loginTrackerMutex.Lock()

	loginTracker[email] = struct{}{}

	return db.User{Email: email}, nil
}
