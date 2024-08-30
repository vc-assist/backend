package restyutil

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

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
