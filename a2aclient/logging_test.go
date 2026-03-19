// Copyright 2025 The A2A Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package a2aclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/log"
)

func TestClientLoggingInterceptorBefore(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := log.AttachLogger(t.Context(), logger)

	interceptor := NewLoggingInterceptor()
	req := &Request{Method: "SendMessage", BaseURL: "http://localhost:9001/invoke"}

	ctx, result, err := interceptor.Before(ctx, req)
	if err != nil {
		t.Fatalf("Before() error = %v", err)
	}
	if result != nil {
		t.Fatalf("Before() result = %v, want nil", result)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v, raw: %s", err, buf.String())
	}

	if msg := entry["msg"].(string); msg != "sending request" {
		t.Errorf("msg = %q, want %q", msg, "sending request")
	}

	_ = ctx
}

func TestClientLoggingInterceptorAfterSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := log.AttachLogger(t.Context(), logger)

	interceptor := NewLoggingInterceptor()
	resp := &Response{
		Method:  "SendMessage",
		BaseURL: "http://localhost:9001",
		Payload: &a2a.Task{ID: "task-1", Status: a2a.TaskStatus{State: a2a.TaskStateCompleted}},
	}

	if err := interceptor.After(ctx, resp); err != nil {
		t.Fatalf("After() error = %v", err)
	}

	if !strings.Contains(buf.String(), "response received") {
		t.Errorf("log = %q, want to contain %q", buf.String(), "response received")
	}
}

func TestClientLoggingInterceptorAfterError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := log.AttachLogger(t.Context(), logger)

	interceptor := NewLoggingInterceptor()
	resp := &Response{
		Method: "GetTask",
		Err:    fmt.Errorf("connection refused"),
	}

	if err := interceptor.After(ctx, resp); err != nil {
		t.Fatalf("After() error = %v", err)
	}

	if !strings.Contains(buf.String(), "WARN") {
		t.Errorf("errors should be logged at WARN level, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "request failed") {
		t.Errorf("log = %q, want to contain %q", buf.String(), "request failed")
	}
}

func TestClientLoggingInterceptorPayloadLevel(t *testing.T) {
	t.Run("info logger does not log payload", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
		ctx := log.AttachLogger(t.Context(), logger)

		interceptor := NewLoggingInterceptor()
		req := &Request{Method: "SendMessage", BaseURL: "http://localhost:9001"}

		_, _, _ = interceptor.Before(ctx, req)

		if strings.Contains(buf.String(), "payload") {
			t.Errorf("payload should not appear at info level: %s", buf.String())
		}
	})

	t.Run("debug logger includes payload", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		ctx := log.AttachLogger(t.Context(), logger)

		interceptor := NewLoggingInterceptor()
		req := &Request{Method: "SendMessage", BaseURL: "http://localhost:9001", Payload: "test-payload"}

		_, _, _ = interceptor.Before(ctx, req)

		if !strings.Contains(buf.String(), "payload") {
			t.Errorf("payload should appear at debug level: %s", buf.String())
		}
	})
}
