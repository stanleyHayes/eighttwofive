// Package email contains EmailSender adapters.
package email

import (
	"context"
	"fmt"
	"html"
	"time"

	"github.com/resend/resend-go/v2"
)

const (
	// maxSendAttempts and sendRetryBackoff bound how hard we retry a failed send.
	// Resend collapses most failures into opaque errors, so we can't reliably
	// tell a transient blip (5xx, network, 429) from a permanent one (invalid
	// recipient, unverified domain); a few short retries recover the transient
	// cases while a permanent failure still surfaces quickly.
	maxSendAttempts  = 3
	sendRetryBackoff = 200 * time.Millisecond
)

// ResendSender sends transactional email through Resend.
type ResendSender struct {
	client *resend.Client
	from   string
}

// NewResendSender builds a sender using the given API key and from address.
func NewResendSender(apiKey, from string) *ResendSender {
	return &ResendSender{client: resend.NewClient(apiKey), from: from}
}

// SendLoginLink sends the single-use sign-in link.
func (s *ResendSender) SendLoginLink(ctx context.Context, to, link string) error {
	err := s.send(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Your Eight Two Five sign-in link",
		Html: `<p>Tap the link below to sign in. It works once and expires in 15 minutes.</p>` +
			`<p><a href="` + link + `">Sign in to Eight Two Five</a></p>` +
			`<p>If you didn't request this, you can ignore it.</p>`,
	})
	if err != nil {
		return fmt.Errorf("resend send login link: %w", err)
	}

	return nil
}

// SendWelcome sends the waitlist confirmation email.
func (s *ResendSender) SendWelcome(ctx context.Context, to, name string) error {
	err := s.send(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "You're on the Eight Two Five waitlist",
		Html: fmt.Sprintf(
			`<p>Hi %s,</p>`+
				`<p>You're on the list. We'll send one note when the storefront opens — nothing else.</p>`+
				`<p>— Eight Two Five</p>`,
			html.EscapeString(name),
		),
	})
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}

	return nil
}

// SendOrderConfirmation notifies the customer that an order status changed.
func (s *ResendSender) SendOrderConfirmation(ctx context.Context, to, name, ref, status string) error {
	err := s.send(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: fmt.Sprintf("Eight Two Five order %s is %s", ref, status),
		Html: fmt.Sprintf(
			`<p>Hi %s,</p>`+
				`<p>Your order <strong>%s</strong> is now <strong>%s</strong>.</p>`+
				`<p>— Eight Two Five</p>`,
			html.EscapeString(name), html.EscapeString(ref), html.EscapeString(status),
		),
	})
	if err != nil {
		return fmt.Errorf("resend send order confirmation: %w", err)
	}

	return nil
}

// SendOrderStatusUpdate notifies the customer of a customer-facing status change
// with an expected timeframe.
func (s *ResendSender) SendOrderStatusUpdate(
	ctx context.Context, to, name, ref, status, timeframe string,
) error {
	err := s.send(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: fmt.Sprintf("Eight Two Five order %s — %s", ref, status),
		Html: fmt.Sprintf(
			`<p>Hi %s,</p>`+
				`<p>Your order <strong>%s</strong> is now <strong>%s</strong>.</p>`+
				`<p>Timeframe: %s</p>`+
				`<p>— Eight Two Five</p>`,
			html.EscapeString(name), html.EscapeString(ref), html.EscapeString(status), html.EscapeString(timeframe),
		),
	})
	if err != nil {
		return fmt.Errorf("resend send order status update: %w", err)
	}

	return nil
}

// send dispatches one email, retrying a bounded number of times with a short
// backoff. Retries are safe because the worst case is a duplicate of the same
// transactional message (e.g. the same single-use sign-in link).
func (s *ResendSender) send(ctx context.Context, req *resend.SendEmailRequest) error {
	var lastErr error

	for attempt := range maxSendAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("send email: %w", ctx.Err())
			case <-time.After(sendRetryBackoff * time.Duration(attempt)):
			}
		}

		_, lastErr = s.client.Emails.SendWithContext(ctx, req)
		if lastErr == nil {
			return nil
		}
	}

	return fmt.Errorf("send email: %w", lastErr)
}
