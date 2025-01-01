package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	report_resty_request  = "resty.request"
	report_resty_response = "resty.response"
)

type instrumentResty struct {
	tel       API
	idcounter *uint64
}

func InstrumentResty(client *resty.Client, tel API) {
	var idcounter uint64
	i := instrumentResty{tel: tel, idcounter: &idcounter}

	client.OnBeforeRequest(i.onBeforeRequest)
	client.OnAfterResponse(i.onAfterResponse)
	client.OnError(i.onError)
}

type reqCtxKeyType int

var reqCtxKey reqCtxKeyType

type reqCtx struct {
	id uint64
	// startTime does not need to rely on chrono because it does not depend on the
	// absolute time, just the difference in time, which can be guaranteed to work.
	startTime time.Time
}

func (i instrumentResty) onBeforeRequest(_ *resty.Client, req *resty.Request) error {
	start := time.Now()
	ctx := req.Context()

	id := atomic.AddUint64(i.idcounter, 1)
	ctx = context.WithValue(ctx, reqCtxKey, reqCtx{
		id:        id,
		startTime: start,
	})
	i.tel.ReportDebug(report_resty_request, id, req.Method, req.URL)

	req.SetContext(ctx)
	return nil
}

func (i instrumentResty) onAfterResponse(_ *resty.Client, res *resty.Response) error {
	end := time.Now()
	ctx := res.Request.Context()

	reqCtx, ok := ctx.Value(reqCtxKey).(reqCtx)
	if !ok {
		panic("failed to get request context")
	}

	duration := end.Sub(reqCtx.startTime)

	i.tel.ReportDebug(
		report_resty_response,
		reqCtx.id,
		duration.String(),
		res.Status(),
	)

	return nil
}

func (i instrumentResty) onError(req *resty.Request, err error) {
	end := time.Now()
	ctx := req.Context()

	reqCtx, ok := ctx.Value(reqCtxKey).(reqCtx)
	if !ok {
		panic("failed to get request context")
	}

	duration := end.Sub(reqCtx.startTime)

	i.tel.ReportBroken(
		report_resty_response,
		err,
		req.Method,
		req.URL,
		duration,
	)
}

func formatHeaders(headers http.Header) string {
	var out strings.Builder
	for k, vals := range headers {
		for _, v := range vals {
			out.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
	rendered := out.String()
	return rendered[:len(rendered)-1]
}

func formatRequestBody(req *http.Request) string {
	body, err := req.GetBody()
	if err != nil {
		return fmt.Sprintf("failed to get request body: %s", err.Error())
	}
	if body == nil {
		return "<NO BODY AVAILABLE>"
	}
	readBody, err := io.ReadAll(body)
	if err != nil {
		return fmt.Sprintf("failed to read request body: %s", err.Error())
	}
	return string(readBody)
}

// 1: request method
// 2: request url
// 3: request headers in ("Key: Value" format)
// 4: request body
// 5: response status
// 6: response url
// 7: response headers in ("Key: Value" format)
// 8: response body
const messageInfoTemplate = `---- REQUEST ----

%s %s

%s

%s

---- RESPONSE ----

%s %s

%s

%s`

func formatHttpMessage(res *resty.Response) string {
	requestHeaders := formatHeaders(res.Request.RawRequest.Header)
	responseHeaders := formatHeaders(res.Header())

	responseUrl := res.Request.URL
	redirected, err := res.RawResponse.Location()
	if err == nil {
		responseUrl = redirected.String()
	}

	return fmt.Sprintf(
		messageInfoTemplate,

		res.Request.Method, res.Request.URL,
		// to trim the last newline off the end of the req headers
		requestHeaders,
		formatRequestBody(res.Request.RawRequest),

		strconv.Itoa(res.StatusCode()), responseUrl,
		responseHeaders,
		res.String(),
	)
}

// 1: request method
// 2: request url
// 3: request headers in ("Key: Value" format)
// 4: request body
const requestInfoTemplate = `---- REQUEST ----

%s %s

%s

%s`

func formatHttpRequest(req *resty.Request) string {
	requestHeaders := formatHeaders(req.Header)
	return fmt.Sprintf(
		requestInfoTemplate,
		req.Method,
		req.URL,
		requestHeaders,
		formatRequestBody(req.RawRequest),
	)
}
