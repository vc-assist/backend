package serviceutil

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Returns a context that will live until Ctrl+C is pressed
func SignalContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	return ctx
}

func StartHttpServer(port int, mux *http.ServeMux) {
	slog.Info("listening to gRPC...", "port", port)
	err := http.ListenAndServe(
		fmt.Sprintf("0.0.0.0:%d", port),
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		Fatal(fmt.Sprintf("failed to listen on port %d", port), err)
	}
}

func Fatal(message string, err error) {
	slog.Error(message, "err", err.Error())
	os.Exit(1)
}

func ProvideAccessTokenInterceptor(accessToken string) connect.UnaryInterceptorFunc {
	authHeader := fmt.Sprintf("Bearer %s", accessToken)
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", authHeader)
			return next(ctx, req)
		}
	}
}

func VerifyAccessTokenInterceptor(accessToken string) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		if accessToken == "" {
			return next
		}
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token := strings.Split(req.Header().Get("Authorization"), " ")
			if len(token) != 2 || token[1] != accessToken {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("Unauthorized"))
			}
			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)

}
func NewConnectOtelInterceptor() *otelconnect.Interceptor {
	otelIntercept, err := otelconnect.NewInterceptor(
		otelconnect.WithTrustRemote(),
		otelconnect.WithoutServerPeerAttributes(),
	)
	if err != nil {
		Fatal("failed to initialize otel interceptor", err)
	}
	return otelIntercept
}
