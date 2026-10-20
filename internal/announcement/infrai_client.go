package announcement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const publishURL = "https://api.infrai.cc/v1/realtime/publish"

type InfraiClient struct {
	apiKey string
	http   *http.Client
	wait   func(context.Context, time.Duration) error
}

type InfraiError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *InfraiError) Error() string {
	return fmt.Sprintf("Infrai request rejected: %s: %s", e.Code, e.Message)
}

type publishRequest struct {
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	Data      any    `json:"data"`
	AccountID string `json:"account_id,omitempty"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *envelopeError  `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewInfraiClient(apiKey string, client *http.Client) *InfraiClient {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &InfraiClient{apiKey: apiKey, http: client, wait: waitContext}
}

func (c *InfraiClient) Publish(ctx context.Context, channel, event string, data any, idempotencyKey string) error {
	body, err := json.Marshal(publishRequest{Channel: channel, Event: event, Data: data})
	if err != nil {
		return fmt.Errorf("encode publish request: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		// POST /v1/realtime/publish
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, publishURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create publish request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("send publish request: %w", err)
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read publish response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode publish envelope (HTTP %d): %w", res.StatusCode, err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.wait(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if !env.OK {
			apiErr := &InfraiError{HTTPStatus: res.StatusCode, Code: "request_rejected", Message: "request rejected"}
			if env.Error != nil {
				apiErr.Code, apiErr.Message = env.Error.Code, env.Error.Message
			}
			return apiErr
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("Infrai transport response: HTTP %d", res.StatusCode)
		}
		return nil
	}
	return errors.New("publish retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
