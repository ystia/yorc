package rest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/armon/go-metrics"
	"github.com/ystia/yorc/helper/metricsutil"
	"github.com/ystia/yorc/log"
)

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				writeError(w, r, newInternalServerError(err))
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Debugf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func acceptHandler(cType string) func(http.Handler) http.Handler {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Accept") != cType {
				writeError(w, r, newNotAcceptableError(cType))
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}

	return m
}

func contentTypeHandler(cType string) func(http.Handler) http.Handler {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") != cType {
				writeError(w, r, newUnsupportedMediaTypeError(cType))
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
	return m
}

type statusRecorderResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorderResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func telemetryHandler(next http.Handler) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {
		var endpointPath string
		if len(r.URL.Path) <= 1 {
			endpointPath = "-"
		} else {
			endpointPath = r.URL.Path[1:]
		}
		defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"http", r.Method, endpointPath}), time.Now())
		writer := &statusRecorderResponseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"http", fmt.Sprint(writer.status), r.Method, endpointPath}), 1)
	}

	return http.HandlerFunc(fn)
}
