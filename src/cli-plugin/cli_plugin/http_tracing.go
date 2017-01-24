package cli_plugin

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
	"time"
)

type tracingRoundTripper struct {
	inner  http.RoundTripper
	writer io.Writer
}

func (t *tracingRoundTripper) dumpRequest(req *http.Request) {
	shouldDisplayBody := !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data")
	dumpedRequest, err := httputil.DumpRequest(req, shouldDisplayBody)
	if err != nil {
		fmt.Fprintf(t.writer, "Error dumping request\n{{.Err}}\n", map[string]interface{}{"Err": err})
	} else {
		fmt.Fprintf(t.writer, "\n%s [%s]\n%s\n", "REQUEST:", time.Now().Format(time.RFC3339), sanitize(string(dumpedRequest)))
		if !shouldDisplayBody {
			fmt.Fprintln(t.writer, "[MULTIPART/FORM-DATA CONTENT HIDDEN]")
		}
	}
}

func (t *tracingRoundTripper) dumpResponse(res *http.Response) {
	dumpedResponse, err := httputil.DumpResponse(res, true)
	if err != nil {
		fmt.Fprintf(t.writer, "Error dumping response\n{{.Err}}\n", map[string]interface{}{"Err": err})
	} else {
		fmt.Fprintf(t.writer, "\n%s [%s]\n%s\n", "RESPONSE:", time.Now().Format(time.RFC3339), sanitize(string(dumpedResponse)))
	}
}

func (t *tracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.writer != nil {
		t.dumpRequest(req)
	}
	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if t.writer != nil {
		t.dumpResponse(resp)
	}
	return resp, nil
}

func wrapWithHTTPTracing(inner http.RoundTripper, tracingEnabled bool) http.RoundTripper {
	if tracingEnabled {
		return &tracingRoundTripper{
			inner:  inner,
			writer: os.Stdout,
		}
	} else {
		return &tracingRoundTripper{
			inner:  inner,
			writer: nil,
		}
	}
}

const PrivateDataPlaceholder = "[PRIVATE DATA HIDDEN]"

func sanitize(input string) string {
	re := regexp.MustCompile(`(?m)^Authorization: .*`)
	sanitized := re.ReplaceAllString(input, "Authorization: "+PrivateDataPlaceholder)

	re = regexp.MustCompile(`password=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "password="+PrivateDataPlaceholder+"&")

	sanitized = sanitizeJSON("token", sanitized)
	sanitized = sanitizeJSON("password", sanitized)

	return sanitized
}

func sanitizeJSON(propertySubstring string, json string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`(?i)"([^"]*%s[^"]*)":\s*"[^\,]*"`, propertySubstring))
	return regex.ReplaceAllString(json, fmt.Sprintf(`"$1":"%s"`, PrivateDataPlaceholder))
}
