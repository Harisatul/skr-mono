package leaderboard

import (
	"context"
	"errors"
	"log/slog"
	"mono-test/feature/shared"
	"mono-test/pkg"
	"net/http"
	"strconv"
	"time"
)

func HttpRoute(mux *http.ServeMux) {
	//mux.Handle("GET /api/http/get", Timeout(5*time.Second)(http.HandlerFunc(getGrade)))
	mux.HandleFunc("GET /api/http/get", getGrade)
}

func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	var lf = []slog.Attr{
		pkg.LogEventName("leaderboard"),
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

func getGrade(w http.ResponseWriter, r *http.Request) {
	var (
		lvState1       = shared.LogEventStateDecodeRequest
		lfState1Status = "state_1_decode_request_status"

		lvState2       = shared.LogEventStateFetchDB
		lfState2Status = "state_2_fetch_grade"

		//ctx = r.Context()
		//ctx, _ = context.WithTimeout(r.Context(), 3*time.Second)
		ctx = pkg.TraceSpanStart(r.Context(), "http.leaderboardHandler")

		lf = []slog.Attr{
			pkg.LogEventName("get-leaderboard"),
		}
	)
	defer pkg.TraceSpanFinish(ctx)

	/*------------------------------------
	| Step 1 : Decode request
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState1))

	var req leaderboardReq
	// Retrieve the query parameter
	tryoutID := r.URL.Query().Get("tid")
	sizeStr := r.URL.Query().Get("size")
	pageStr := r.URL.Query().Get("page")

	if tryoutID == "" {
		pkg.TraceSpanError(ctx, errors.New("missing id query parameter"))
		lf = append(lf, pkg.LogStatusFailed(lfState1Status))
		pkg.LogWarnWithContext(ctx, "missing id query parameter", nil, lf)
		shared.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing id query parameter"))
		return
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState1Status))
		pkg.LogWarnWithContext(ctx, "failed to cast size query param", nil, lf)
		shared.WriteErrorResponse(w, http.StatusBadRequest, errors.New("failed to cast size query param"))
		return
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState1Status))
		pkg.LogWarnWithContext(ctx, "failed to cast page query param", nil, lf)
		shared.WriteErrorResponse(w, http.StatusBadRequest, errors.New("failed to cast page query param"))
		return
	}

	lf = append(lf,
		pkg.LogStatusSuccess(lfState1Status),
		pkg.LogEventPayload(req),
	)
	/*------------------------------------
	| Step 2 : Fetch Leaderboard
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState2))

	offset := (page - 1) * size

	grade, err := fetchGrade(ctx, tryoutID, size, offset)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}
	lf = append(lf, pkg.LogStatusSuccess(lfState2Status))

	shared.WriteSuccessResponse(w, http.StatusOK,
		grade,
	)
	pkg.LogInfoWithContext(ctx, "success get leaderboard", lf)
}
