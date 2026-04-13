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

const maxLoggedBody = 4096

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := slogctx.FromCtx(r.Context())

		args := requestArgs(r)
		log.With(args...).Debug("⚔️  HTTP START  "+r.Method+" "+r.URL.Path, "header", r.Header)

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

		args = append(requestArgs(r), "duration", time.Since(tmStart).String())

		if lrw.wroteHeader {
			args = append(args,
				"status", status,
				"response_size", humanize.Bytes(uint64(lrw.size)))
			if lrw.body.Len() > 0 {
				args = append(args, "body", sanitizeForLog(lrw.body.String()))
			}
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
	status      int
	size        int
	wroteHeader bool
	body        bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if lrw.wroteHeader {
		return
	}
	lrw.wroteHeader = true
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if !lrw.wroteHeader {
		lrw.WriteHeader(http.StatusOK)
	}
	if lrw.status >= 400 {
		// Сохраняем только первые N байт, чтобы не тащить в лог огромные ответы.
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
