package tryout

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"log"
	"log/slog"
	"mono-test/feature/shared"
	"mono-test/pkg"
	"net/http"
	"time"
)

func HttpRoute(mux *http.ServeMux) {
	//mux.Handle("POST /api/submission/create", Timeout(5*time.Second)(http.HandlerFunc(insertSubmissionHandler)))
	mux.HandleFunc("POST /api/submission/create", insertSubmissionHandler)
	//mux.Handle("POST /api/answer/create", Timeout(5*time.Second)(http.HandlerFunc(insertAnswerHandler)))
	mux.HandleFunc("POST /api/answer/create", insertAnswerHandler)
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
		pkg.LogEventName("exam-http-handler"),
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("Request processed in %s", duration)
	})
}

func insertSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	var (
		lvState1       = shared.LogEventStateDecodeRequest
		lfState1Status = "state_1_decode_request_status"

		lvState2       = shared.LogEventStateInsertDB
		lfState2Status = "state_2_insert_submission`"

		//ctx = r.Context()
		ctx = pkg.TraceSpanStart(r.Context(), "http.examHandler")

		lf = []slog.Attr{
			pkg.LogEventName("insert-submission"),
		}
	)
	defer pkg.TraceSpanFinish(ctx)

	/*------------------------------------
	| Step 1 : Decode request
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState1))

	var req submissionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState1Status))
		pkg.LogWarnWithContext(ctx, "invalid request", err, lf)
		shared.WriteErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	lf = append(lf,
		pkg.LogStatusSuccess(lfState1Status),
		pkg.LogEventPayload(req),
	)
	/*------------------------------------
	| Step 2 : Insert Submission
	* ----------------------------------*/

	id, err := pkg.GenerateId()
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	lf = append(lf, pkg.LogEventState(lvState2))

	parse, err := uuid.Parse(req.TryoutID)
	if err != nil {
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	submission := UserTestSubmission{
		ID:       id,
		Token:    req.Token,
		TryoutID: parse,
	}

	sid, _, _, _, err := insertSubmission(ctx, submission)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}
	lf = append(lf, pkg.LogStatusSuccess(lfState2Status))
	shared.WriteSuccessResponse(w, http.StatusOK,
		submissionResponse{
			Id: sid,
		},
	)
	pkg.LogInfoWithContext(ctx, "success insert submission", lf)
}

func insertAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var (
		lvState1       = shared.LogEventStateDecodeRequest
		lfState1Status = "state_1_decode_request_status"

		lvState2       = shared.LogEventStateInsertDB
		lfState2Status = "state_2_insert_answer`"

		lvState3       = shared.LogEventStateFetchDB
		lfState3Status = "state_3_fetch_db_status"

		lvState4       = shared.LogEventStateUpdateDB
		lfState4Status = "state_3_update_db_status"

		//ctx = r.Context()
		ctx = pkg.TraceSpanStart(r.Context(), "http.answerHandler")

		lf = []slog.Attr{
			pkg.LogEventName("insert-answer"),
		}
	)
	defer pkg.TraceSpanFinish(ctx)

	/*------------------------------------
	| Step 1 : Decode request
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState1))

	var req answerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState1Status))
		pkg.LogWarnWithContext(ctx, "invalid request", err, lf)
		shared.WriteErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	lf = append(lf,
		pkg.LogStatusSuccess(lfState1Status),
		pkg.LogEventPayload(req),
	)
	/*------------------------------------
	| Step 2 : Insert Answer
	* ----------------------------------*/
	tx, err := db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadUncommitted,
	})
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	id, err := pkg.GenerateId()
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	lf = append(lf, pkg.LogEventState(lvState2))

	sparse, err := uuid.Parse(req.SubmissionID)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	qparse, err := uuid.Parse(req.QuestionID)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	cparse, err := uuid.Parse(req.ChoiceID)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	answer := UserSubmittedAnswer{
		ID:                   id,
		QuestionID:           qparse,
		ChoiceID:             cparse,
		UserTestSubmissionID: sparse,
	}

	aid, _, choiceId, sid, err := insertAnswer(ctx, tx, answer)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}
	lf = append(lf, pkg.LogStatusSuccess(lfState2Status))

	/*------------------------------------
	| Step 3 : Fetch Choice Data
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState3))

	idr, err := fetchChoices(ctx, tx, choiceId.String())
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		lf = append(lf, pkg.LogStatusFailed(lfState3Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		return
	}

	lf = append(lf, pkg.LogStatusSuccess(lfState3Status))

	/*------------------------------------
	| Step 4 : Update Submission Data
	* ----------------------------------*/
	lf = append(lf, pkg.LogEventState(lvState4))

	if idr == true {
		_, _, _, _, err = updateSubmission(ctx, tx, sid.String())
		if err != nil {
			pkg.TraceSpanError(ctx, err)
			lf = append(lf, pkg.LogStatusFailed(lfState4Status))
			pkg.LogErrorWithContext(ctx, err, lf)
			return
		}
	} else {
		_, _, _, _, err = updateSubmissionW(ctx, tx, sid.String())
		if err != nil {
			pkg.TraceSpanError(ctx, err)
			lf = append(lf, pkg.LogStatusFailed(lfState4Status))
			pkg.LogErrorWithContext(ctx, err, lf)
			return
		}
	}
	lf = append(lf, pkg.LogStatusSuccess(lfState3Status))

	err = tx.Commit(ctx)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		_ = tx.Rollback(ctx)
		lf = append(lf, pkg.LogStatusFailed(lfState2Status))
		pkg.LogErrorWithContext(ctx, err, lf)
		shared.WriteInternalServerErrorResponse(w)
		return
	}

	shared.WriteSuccessResponse(w, http.StatusOK,
		answerResponse{
			Id: aid,
		},
	)
	pkg.LogInfoWithContext(ctx, "success get tryout", lf)
}
