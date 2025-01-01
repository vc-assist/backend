package main

import (
	"encoding/json"
	"net/http"
	"os"
	gradestoredb "vcassist-backend/lib/gradestore/db"
	"vcassist-backend/pkg/migrations"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/proto/vcassist/services/sis/v1/sisv1connect"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/vcsis"
	vcsisdb "vcassist-backend/services/vcsis/db"

	"connectrpc.com/connect"
)

type VCSisOAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type VCSisConfig struct {
	Database           string           `json:"database"`
	PowerschoolBaseUrl string           `json:"powerschool_base_url"`
	PowerschoolOAuth   VCSisOAuthConfig `json:"powerschool_oauth"`
	WeightsFile        string           `json:"weights_file"`
}

func InitVCSis(
	mux *http.ServeMux,
	verify verifier.Verifier,
	cfg VCSisConfig,
	keychain keychainv1connect.KeychainServiceClient,
	linker linkerv1connect.LinkerServiceClient,
) error {
	database, err := migrations.OpenDB(
		vcsisdb.Schema+"\n"+gradestoredb.Schema,
		cfg.Database,
	)
	if err != nil {
		return err
	}

	var weights vcsis.WeightData
	if cfg.WeightsFile != "" {
		buff, err := os.ReadFile(cfg.WeightsFile)
		if err != nil {
			return err
		}
		err = json.Unmarshal(buff, &weights)
		if err != nil {
			return err
		}
	}

	sisv1connect.SIServiceTracer = telemetry.Tracer("vcsis")
	mux.Handle(sisv1connect.NewSIServiceHandler(
		sisv1connect.NewInstrumentedSIServiceClient(
			vcsis.NewService(
				vcsis.ServiceOptions{
					Database:   database,
					Keychain:   keychain,
					Linker:     linker,
					BaseUrl:    cfg.PowerschoolBaseUrl,
					OAuth:      vcsis.OAuthConfig(cfg.PowerschoolOAuth),
					WeightData: weights,
				},
			),
		),
		connect.WithInterceptors(
			verifier.NewAuthInterceptor(verify),
		),
	))
	return nil
}
