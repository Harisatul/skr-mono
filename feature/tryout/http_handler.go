package tryout

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"mono-test/feature/shared"
	"mono-test/pkg"
	"net/http"
	"time"
)

func HttpRoute(mux *http.ServeMux) {
	//mux.Handle("GET /api/tryout/detail", Timeout(5*time.Second)(http.HandlerFunc(detailTryoutHandler)))
	mux.HandleFunc("GET /api/tryout/detail", detailTryoutHandler)
	mux.Handle("GET /api/metrics", promhttp.Handler())
}

//func timeoutMiddleware(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
//		defer cancel()
//
//		r = r.WithContext(ctx)
//
//		done := make(chan struct{})
//		go func() {
//			next.ServeHTTP(w, r)
//			close(done)
//		}()
//
//		select {
//		case <-done:
//			return
//		case <-ctx.Done():
//			w.WriteHeader(http.StatusGatewayTimeout)
//			w.Write([]byte("Request timed out"))
//		}
//	})
//}

func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	var lf = []slog.Attr{
		pkg.LogEventName("DetailTryout"),
	}
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer func() {
				cancel()
				if ctx.Err() == context.DeadlineExceeded {
					pkg.LogErrorWithContext(ctx, ctx.Err(), lf)
					shared.WriteInternalServerErrorResponse(w)
				}
			}()
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

var (
	requestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "detail_tryout_requests_total",
			Help: "Total number of detail tryout requests",
		},
		[]string{"status"},
	)
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "detail_tryout_request_duration_seconds",
			Help:    "Duration of detail tryout requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)
)

func detailTryoutHandler(w http.ResponseWriter, r *http.Request) {
	var (
		lvState1       = shared.LogEventStateDecodeRequest
		lfState1Status = "state_1_decode_request_status"

		lvState2       = shared.LogEventStateFetchDB
		lfState2Status = "state_2_get_detail_tryout"

		//ctx = r.Context()
		ctx = pkg.TraceSpanStart(r.Context(), "http.getTryoutHandler")

		lf = []slog.Attr{
			pkg.LogEventName("DetailTryout"),
		}
	)

	// Start a timer to measure request duration
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		status := "success"
		if v > 0 {
			status = "failure"
		}
		requestDuration.WithLabelValues(status).Observe(v)
	}))
	defer timer.ObserveDuration()

	defer pkg.TraceSpanFinish(ctx)

	/*------------------------------------
	| Step 1 : Decode request
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState1))

	var req detailTryoutRequest
	// Retrieve the query parameter
	tryoutID := r.URL.Query().Get("id")
	if tryoutID == "" {
		pkg.TraceSpanError(ctx, errors.New("missing id query parameter"))
		lf = append(lf, pkg.LogStatusFailed(lfState1Status))
		pkg.LogWarnWithContext(ctx, "missing id query parameter", nil, lf)
		shared.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing id query parameter"))
		requestCount.WithLabelValues("failure").Inc()
		return
	}
	req.ID = tryoutID

	lf = append(lf,
		pkg.LogStatusSuccess(lfState1Status),
		pkg.LogEventPayload(req),
	)

	/*------------------------------------
	| Step 2 : GET TRYOUT
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState2))

	tryout, err := getTryoutWithQuestions(ctx, req.ID)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		requestCount.WithLabelValues("failure").Inc()
		return
	}

	lf = append(lf, pkg.LogStatusSuccess(lfState2Status))
	shared.WriteSuccessResponse(w, http.StatusOK,
		tryout,
	)
	requestCount.WithLabelValues("success").Inc()
	pkg.LogInfoWithContext(ctx, "success get tryout", lf)
}
