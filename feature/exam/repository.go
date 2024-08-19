package tryout

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"mono-test/pkg"
	"time"
)

func insertSubmission(ctx context.Context, submission UserTestSubmission) (id uuid.UUID, version int64, token string, tryoutid uuid.UUID, err error) {
	ctx = pkg.TraceSpanStart(ctx, "repo.inserSubmission")
	defer pkg.TraceSpanFinish(ctx)

	// Prepare the SQL insert statement
	sql := `INSERT INTO submission (id, created_at, updated_at, submission_start_time, version, submission_end_time, token, tryout_id, score) 
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, version, token, tryout_id`

	// Execute the insert statement
	err = db.QueryRow(ctx, sql, submission.ID, time.Now().UnixMilli(), time.Now().UnixMilli(), time.Now(), 1, nil, submission.Token, submission.TryoutID, 0).Scan(&id, &version, &token, &tryoutid)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		if err == pgx.ErrNoRows {
			err = nil
		}
	}
	return
}

func insertAnswer(ctx context.Context, tx pgx.Tx, answer UserSubmittedAnswer) (id uuid.UUID, version int64, choice_id uuid.UUID, user_test_submission_id uuid.UUID, err error) {
	ctx = pkg.TraceSpanStart(ctx, "repo.insertAnswer")
	defer pkg.TraceSpanFinish(ctx)

	sql := `INSERT INTO answer_submitted (id, created_at, updated_at, question_id, choice_id, user_test_submission_id, version) 
            VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, version, choice_id, user_test_submission_id`

	// Execute the insert statement
	err = tx.QueryRow(ctx, sql, answer.ID, time.Now().UnixMilli(), time.Now().UnixMilli(), answer.QuestionID, answer.ChoiceID, answer.UserTestSubmissionID, 1).Scan(&id, &version, &choice_id, &user_test_submission_id)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		if err == pgx.ErrNoRows {
			err = nil
		}
	}
	return
}

func fetchChoices(ctx context.Context, tx pgx.Tx, id string) (idr bool, err error) {
	ctx = pkg.TraceSpanStart(ctx, "repo.fetchChoices")
	defer pkg.TraceSpanFinish(ctx)
	query := "SELECT is_correct FROM choices WHERE id = $1"

	// Execute the insert statement
	err = tx.QueryRow(ctx, query, id).Scan(&idr)
	if err != nil {
		pkg.TraceSpanError(ctx, err)
		if err == pgx.ErrNoRows {
			err = nil
		}
	}
	return
}

func updateSubmission(ctx context.Context, tx pgx.Tx, submission string) (id uuid.UUID, version int64, token string, score int, err error) {
	ctx = pkg.TraceSpanStart(ctx, "repo.updateScoreCorrect")
	defer pkg.TraceSpanFinish(ctx)
	// Prepare the SQL update statement
	sql := `
		UPDATE submission 
		SET updated_at = $1, version = version + 1,  score = score + 5
		WHERE id = $2
		RETURNING id, version, token, score
	`

	// Execute the update statement
	err = tx.QueryRow(ctx, sql, time.Now().UnixMilli(), submission).Scan(&id, &version, &token, &score)

	if err != nil {
		pkg.TraceSpanError(ctx, err)
		if err == pgx.ErrNoRows {
			err = nil
		}
	}
	return
}

func updateSubmissionW(ctx context.Context, tx pgx.Tx, submission string) (id uuid.UUID, version int64, token string, score int, err error) {
	ctx = pkg.TraceSpanStart(ctx, "repo.updateScoreIncorrect")
	defer pkg.TraceSpanFinish(ctx)
	// Prepare the SQL update statement
	sql := `
		UPDATE submission 
		SET updated_at = $1, version = version + 1,  score = score + 0
		WHERE id = $2
		RETURNING id, version, token, score
	`

	// Execute the update statement
	err = tx.QueryRow(ctx, sql, time.Now().UnixMilli(), submission).Scan(&id, &version, &token, &score)

	if err != nil {
		pkg.TraceSpanError(ctx, err)
		if err == pgx.ErrNoRows {
			err = nil
		}
	}
	return
}
