package store

import (
	"context"
	"database/sql"
)

func addPaymentSubmittedNotification(ctx context.Context, tx *sql.Tx, userID, paymentID int64, paymentType string, jobID *int64, now int64) error {
	title, message := "Hiring pass payment submitted", "Your ₱10 receipt was submitted and is waiting for accounting verification."
	if paymentType == "employer" {
		title = "Search fee payment submitted"
		message = "Your search fee receipt was submitted and is waiting for accounting verification."
	}
	var payloadJobID any
	if jobID != nil {
		payloadJobID = *jobID
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		VALUES(?,?,?,'payment',json_object('payment_type',?,'payment_id',?,'job_id',?),?)`,
		userID, title, message, paymentType, paymentID, payloadJobID, now)
	return err
}
