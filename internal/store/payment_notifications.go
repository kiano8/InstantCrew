package store

import (
	"context"
	"database/sql"
)

func addPaymentSubmittedNotification(ctx context.Context, tx *sql.Tx, userID, paymentID int64, paymentType string, jobID *int64, now int64) error {
	title, message := "Hiring pass payment submitted", "Your hiring pass receipt was submitted and is waiting for accounting verification."
	if paymentType == "employer" {
		title = "Search fee payment submitted"
		message = "Your search fee receipt was submitted and is waiting for accounting verification."
	}
	var payloadJobID any
	if jobID != nil {
		payloadJobID = *jobID
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		VALUES($1,$2,$3,'payment',json_build_object('payment_type',$4::text,'payment_id',$5::bigint,'job_id',$6::bigint)::text,$7)`,
		userID, title, message, paymentType, paymentID, payloadJobID, now)
	return err
}
