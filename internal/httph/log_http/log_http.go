package log_http

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	slogctx "github.com/veqryn/slog-context"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := slogctx.FromCtx(r.Context())

		args := requestArgs(r)
		reqHeaders := filterHeaders(r.Header, requestHeaderWhitelist)
		if len(reqHeaders) > 0 {
			args = append(args, "request_headers", reqHeaders)
		}

		log.With(args...).Debug("⚔️  HTTP START  " + r.Method + " " + r.URL.Path)

		tmStart := time.Now()

		lrw := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(lrw, r)

		status := lrw.status
		if status == 0 {
			status = http.StatusOK
		}

		logLevel := slog.LevelDebug
		if status >= 500 {
			logLevel = slog.LevelError
		} else if status >= 400 {
			logLevel = slog.LevelWarn
		}

		args = append(requestArgs(r),
			"duration", time.Since(tmStart).String(),
			"status", status,
			"response_size", humanize.Bytes(uint64(lrw.size)),
		)

		if respHeaders := filterHeaders(lrw.responseHeaders(), responseHeaderWhitelist); len(respHeaders) > 0 {
			args = append(args, "response_headers", respHeaders)
		}

		if lrw.body.Len() > 0 {
			args = append(args, "body", sanitizeForLog(lrw.body.String()))
		}

		log.With(args...).Log(r.Context(), logLevel, "⚔️  HTTP FINISH "+r.Method+" "+r.URL.Path)
	})
}

func requestArgs(r *http.Request) (args []any) {
	if len(r.URL.Query()) != 0 {
		args = append(args, "query", r.URL.Query())
	}
	return args
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status                 int
	size                   int
	wroteHeader            bool
	body                   bytes.Buffer
	capturedResponseHeader http.Header
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if lrw.wroteHeader {
		return
	}
	lrw.wroteHeader = true
	lrw.status = code
	lrw.capturedResponseHeader = cloneHeader(lrw.ResponseWriter.Header())
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if !lrw.wroteHeader {
		lrw.WriteHeader(http.StatusOK)
	}
	if lrw.status >= 400 {
		remain := maxLoggedBody - lrw.body.Len()
		if remain > 0 {
			if len(b) > remain {
				lrw.body.Write(b[:remain])
			} else {
				lrw.body.Write(b)
			}
		}
	}

	n, err := lrw.ResponseWriter.Write(b)
	lrw.size += n
	return n, err
}

func (lrw *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return lrw.ResponseWriter
}

func (lrw *loggingResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (lrw *loggingResponseWriter) responseHeaders() http.Header {
	if lrw.capturedResponseHeader != nil {
		return lrw.capturedResponseHeader
	}
	return cloneHeader(lrw.ResponseWriter.Header())
}

func filterHeaders(h http.Header, whitelist map[string]struct{}) http.Header {
	if len(h) == 0 {
		return nil
	}

	out := make(http.Header)
	for k, vv := range h {
		ck := http.CanonicalHeaderKey(k)
		if _, ok := whitelist[ck]; !ok {
			continue
		}
		out[ck] = append([]string(nil), vv...)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneHeader(h http.Header) http.Header {
	if len(h) == 0 {
		return nil
	}
	cp := make(http.Header, len(h))
	for k, vv := range h {
		cp[k] = append([]string(nil), vv...)
	}
	return cp
}

func sanitizeForLog(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.Trim(s, "\"")
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

const maxLoggedBody = 4096

var requestHeaderWhitelist = map[string]struct{}{
	"Accept":           {},
	"Accept-Encoding":  {},
	"Authorization":    {},
	"Content-Type":     {},
	"User-Agent":       {},
	"X-Correlation-Id": {},
	"Traceparent":      {},
	"Tracestate":       {},
}

var responseHeaderWhitelist = map[string]struct{}{
	"Content-Type":                 {},
	"Content-Length":               {},
	"Location":                     {},
	"Retry-After":                  {},
	"WWW-Authenticate":             {},
	"X-Correlation-Id":             {},
	"Traceparent":                  {},
	"Access-Control-Allow-Origin":  {},
	"Access-Control-Allow-Headers": {},
	"Access-Control-Allow-Methods": {},
}
