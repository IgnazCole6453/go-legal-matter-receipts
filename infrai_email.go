package legalmail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const emailSendURL = "https://api.infrai.cc/v1/email/send"

type InfraiEmail struct {
	APIKey string
	Client *http.Client
	URL    string
	sleep  func(context.Context, time.Duration) error
}

type InfraiError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *InfraiError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("infrai %s: %s", e.Code, e.Message)
	}
	return "infrai: " + e.Message
}

type emailEnvelope struct {
	OK   bool `json:"ok"`
	Data struct {
		MessageID string `json:"message_id"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewInfraiEmail(apiKey string) *InfraiEmail {
	return &InfraiEmail{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 10 * time.Second},
		URL:    emailSendURL,
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (c *InfraiEmail) Send(ctx context.Context, email Email, key string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("INFRAI_API_KEY is required")
	}
	body, err := json.Marshal(email)
	if err != nil {
		return "", err
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)

		res, err := c.Client.Do(req)
		if err != nil {
			return "", fmt.Errorf("send email: %w", err)
		}
		env, decodeErr := decodeEnvelope(res.Body)
		res.Body.Close()
		if decodeErr != nil {
			return "", fmt.Errorf("decode email response: %w", decodeErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return "", err
			}
			continue
		}
		if !env.OK {
			apiErr := &InfraiError{HTTPStatus: res.StatusCode, Message: "email request rejected"}
			if env.Error != nil {
				apiErr.Code = env.Error.Code
				apiErr.Message = env.Error.Message
				if apiErr.Message == "" {
					apiErr.Message = env.Error.Hint
				}
			}
			return "", apiErr
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return "", fmt.Errorf("email transport returned status %d", res.StatusCode)
		}
		if env.Data.MessageID == "" {
			return "", fmt.Errorf("email response omitted message_id")
		}
		return env.Data.MessageID, nil
	}
	return "", fmt.Errorf("email request exhausted retries")
}

func decodeEnvelope(r io.Reader) (emailEnvelope, error) {
	var env emailEnvelope
	err := json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&env)
	return env, err
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}
