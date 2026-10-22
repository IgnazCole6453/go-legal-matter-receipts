package legalmail

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"time"
)

type NoticeKind string

const (
	MatterIntake     NoticeKind = "matter_intake"
	SignedDocument   NoticeKind = "signed_document"
	DeadlineFollowUp NoticeKind = "deadline_follow_up"
)

type Notice struct {
	Kind         NoticeKind `json:"kind"`
	MatterID     string     `json:"matter_id"`
	ClientName   string     `json:"client_name"`
	To           string     `json:"to"`
	DocumentName string     `json:"document_name,omitempty"`
	DownloadURL  string     `json:"download_url,omitempty"`
	Deadline     string     `json:"deadline,omitempty"`
}

type Delivery struct {
	MessageID string `json:"message_id"`
	Notice    string `json:"notice"`
}

type Email struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type EmailSender interface {
	Send(context.Context, Email, string) (string, error)
}

type ReceiptSender struct {
	email EmailSender
	now   func() time.Time
}

func NewReceiptSender(email EmailSender) *ReceiptSender {
	return &ReceiptSender{email: email, now: time.Now}
}

func (s *ReceiptSender) Dispatch(ctx context.Context, n Notice) (Delivery, error) {
	if n.MatterID == "" || n.ClientName == "" || n.To == "" {
		return Delivery{}, fmt.Errorf("matter_id, client_name, and to are required")
	}

	email, label, err := buildEmail(n, s.now())
	if err != nil {
		return Delivery{}, err
	}
	messageID, err := s.email.Send(ctx, email, idempotencyKey(n))
	if err != nil {
		return Delivery{}, err
	}
	return Delivery{MessageID: messageID, Notice: label}, nil
}

func buildEmail(n Notice, now time.Time) (Email, string, error) {
	name := html.EscapeString(n.ClientName)
	matterID := html.EscapeString(n.MatterID)
	switch n.Kind {
	case MatterIntake:
		return Email{
			To:      n.To,
			Subject: fmt.Sprintf("Matter %s intake received", n.MatterID),
			HTML:    fmt.Sprintf("<p>Hello %s,</p><p>We received your intake for matter <strong>%s</strong>.</p>", name, matterID),
		}, "intake receipt", nil
	case SignedDocument:
		if n.DocumentName == "" || n.DownloadURL == "" {
			return Email{}, "", fmt.Errorf("document_name and download_url are required for signed_document")
		}
		return Email{
			To:      n.To,
			Subject: fmt.Sprintf("Signed document ready for matter %s", n.MatterID),
			HTML: fmt.Sprintf("<p>Hello %s,</p><p>Your signed %s is ready: <a href=\"%s\">download document</a>.</p>",
				name, html.EscapeString(n.DocumentName), html.EscapeString(n.DownloadURL)),
		}, "signed document delivery", nil
	case DeadlineFollowUp:
		deadline, err := time.Parse("2006-01-02", n.Deadline)
		if err != nil {
			return Email{}, "", fmt.Errorf("deadline must use YYYY-MM-DD")
		}
		days := int(deadline.Sub(midnight(now)).Hours() / 24)
		if days < 0 || days > 7 {
			return Email{}, "", fmt.Errorf("deadline follow-up is sent only from 0 through 7 days before the deadline")
		}
		return Email{
			To:      n.To,
			Subject: fmt.Sprintf("Matter %s deadline in %d days", n.MatterID, days),
			HTML:    fmt.Sprintf("<p>Hello %s,</p><p>Matter <strong>%s</strong> has a deadline on %s.</p>", name, matterID, deadline.Format("January 2, 2006")),
		}, "deadline follow-up", nil
	default:
		return Email{}, "", fmt.Errorf("kind must be matter_intake, signed_document, or deadline_follow_up")
	}
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func idempotencyKey(n Notice) string {
	sum := sha256.Sum256([]byte(string(n.Kind) + "\x00" + n.MatterID + "\x00" + n.To + "\x00" + n.Deadline + "\x00" + n.DownloadURL))
	return "legal-notice-" + hex.EncodeToString(sum[:16])
}
