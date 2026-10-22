package legalmail

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInfraiEmailRetriesRateLimitAndPreservesBoundary(t *testing.T) {
	requests := 0
	client := NewInfraiEmail("test-key")
	client.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodPost || req.URL.String() != emailSendURL {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" || req.Header.Get("Idempotency-Key") != "notice-42" {
			t.Fatal("missing auth or idempotency header")
		}
		body, _ := io.ReadAll(req.Body)
		if strings.Contains(string(body), "from") || !strings.Contains(string(body), `"subject":"Receipt"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		if requests == 1 {
			return response(http.StatusTooManyRequests, `{"ok":false,"data":{},"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":{}}`, "0"), nil
		}
		return response(http.StatusOK, `{"ok":true,"data":{"message_id":"msg_42"},"error":null,"metadata":{}}`, ""), nil
	})}
	client.sleep = func(context.Context, time.Duration) error { return nil }

	messageID, err := client.Send(context.Background(), Email{To: "client@example.com", Subject: "Receipt", HTML: "<p>Received</p>"}, "notice-42")
	if err != nil {
		t.Fatal(err)
	}
	if messageID != "msg_42" || requests != 2 {
		t.Fatalf("messageID/requests = %q/%d", messageID, requests)
	}
}

func response(status int, body, retryAfter string) *http.Response {
	header := make(http.Header)
	if retryAfter != "" {
		header.Set("Retry-After", retryAfter)
	}
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}
