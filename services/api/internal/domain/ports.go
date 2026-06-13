package domain

import "context"

// EmailSender is the outbound port for transactional email.
type EmailSender interface {
	SendWelcome(ctx context.Context, to, name string) error
	SendLoginLink(ctx context.Context, to, link string) error
	SendOrderConfirmation(ctx context.Context, to, name, ref, status string) error
	// SendOrderStatusUpdate notifies the customer that their order has moved to
	// one of the customer-facing production statuses ("order confirmed",
	// "in production", "ready"), including the expected timeframe.
	SendOrderStatusUpdate(ctx context.Context, to, name, ref, status, timeframe string) error
}

// UploadSignature contains everything a browser needs to upload a file
// directly to the media CDN without proxying bytes through this API.
type UploadSignature struct {
	CloudName string
	APIKey    string
	Timestamp int64
	Folder    string
	Signature string
}

// UploadSigner is the outbound port for signing direct-to-CDN uploads.
type UploadSigner interface {
	SignUpload(folder string) (UploadSignature, error)
}
