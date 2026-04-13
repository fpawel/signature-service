package httph

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-openapi/runtime/middleware"
)

// Use связывает несколько промежуточных обработчиков http в один.
func Use(middlewares ...middleware.Builder) (result middleware.Builder) {
	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

// Recovery промежуточное ПО обработки паники в обработчиках http.
// Регистрирует панику в журнале, прекращает обработку запроса и возвращает ответ с кодом 500.
// canRespondStack разрешает ответ с содержимым стека.
func Recovery(canRespondStack bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				_panic := recover()
				if _panic == nil {
					return
				}
				msg := fmt.Sprintf("panic %s %s", _panic, debug.Stack())
				slog.ErrorContext(r.Context(), msg)
				if !canRespondStack {
					msg = "Произошёл внутренний сбой."
				}
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(msg))
			}()
			h.ServeHTTP(w, r)
		})
	}
}

// SetResponseHeader промежуточное ПО для установки в ответе http заголовка h со значением v.
func SetResponseHeader(h, v string) middleware.Builder {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(NewWriterHeader(w, h, v), r)
		})
	}
}

// writerHeader implements http.ResponseWriter to write predefined header value
type writerHeader struct {
	http.ResponseWriter
	written bool
	header  string
	value   string
}

func NewWriterHeader(w http.ResponseWriter, h, v string) http.ResponseWriter {
	return &writerHeader{
		ResponseWriter: w,
		header:         h,
		value:          v,
	}
}

func (w *writerHeader) WriteHeader(code int) {
	if !w.written {
		w.ResponseWriter.Header().Set(w.header, w.value)
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}
