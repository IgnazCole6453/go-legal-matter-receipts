package legalmail

import (
	"context"
	"strings"
	"testing"
	"time"
)

type recordingEmail struct {
	email Email
	key   string
}

func (r *recordingEmail) Send(_ context.Context, email Email, key string) (string, error) {
	r.email, r.key = email, key
	return "msg_legal_42", nil
}

func TestDispatchBusinessDecision(t *testing.T) {
	tests := []struct {
		name        string
		notice      Notice
		wantLabel   string
		wantSubject string
		wantErr     string
	}{
		{
			name:      "intake becomes a receipt",
			notice:    Notice{Kind: MatterIntake, MatterID: "M-42", ClientName: "Avery", To: "avery@example.com"},
			wantLabel: "intake receipt", wantSubject: "Matter M-42 intake received",
		},
		{
			name:      "signed agreement includes delivery link",
			notice:    Notice{Kind: SignedDocument, MatterID: "M-42", ClientName: "Avery", To: "avery@example.com", DocumentName: "Engagement agreement", DownloadURL: "https://files.example.com/signed/42"},
			wantLabel: "signed document delivery", wantSubject: "Signed document ready for matter M-42",
		},
		{
			name:      "deadline inside seven days is followed up",
			notice:    Notice{Kind: DeadlineFollowUp, MatterID: "M-42", ClientName: "Avery", To: "avery@example.com", Deadline: "2026-09-15"},
			wantLabel: "deadline follow-up", wantSubject: "Matter M-42 deadline in 7 days",
		},
		{
			name:    "distant deadline stays quiet",
			notice:  Notice{Kind: DeadlineFollowUp, MatterID: "M-42", ClientName: "Avery", To: "avery@example.com", Deadline: "2026-09-16"},
			wantErr: "0 through 7 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &recordingEmail{}
			sender := NewReceiptSender(recorder)
			sender.now = func() time.Time { return time.Date(2026, 9, 8, 15, 0, 0, 0, time.UTC) }
			got, err := sender.Dispatch(context.Background(), tt.notice)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Dispatch() error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Notice != tt.wantLabel || recorder.email.Subject != tt.wantSubject {
				t.Fatalf("notice/subject = %q/%q", got.Notice, recorder.email.Subject)
			}
			if recorder.key == "" {
				t.Fatal("missing idempotency key")
			}
		})
	}
}
