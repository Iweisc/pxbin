package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// wrappedWriter captures the status code from WriteHeader.
type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

var writerPool = sync.Pool{
	New: func() any {
		return &wrappedWriter{}
	},
}

func (w *wrappedWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *wrappedWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter for http.Flusher support.
func (w *wrappedWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Flush implements http.Flusher for streaming support.
func (w *wrappedWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Middleware returns chi-compatible middleware that records request metrics.
func Middleware(m *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := writerPool.Get().(*wrappedWriter)
			ww.ResponseWriter = w
			ww.statusCode = http.StatusOK
			ww.written = false

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			statusStr := strconv.Itoa(ww.statusCode)
			path := r.URL.Path

			m.RequestsTotal.WithLabelValues(r.Method, path, statusStr, "").Inc()
			m.RequestDuration.WithLabelValues(r.Method, path).Observe(duration.Seconds())

			ww.ResponseWriter = nil
			writerPool.Put(ww)
		})
	}
}
