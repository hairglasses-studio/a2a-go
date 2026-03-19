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

package a2asrv

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

func TestLoggingInterceptorBefore(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := log.AttachLogger(t.Context(), logger)

	interceptor := NewLoggingInterceptor()
	callCtx := &CallContext{method: "SendMessage"}
	req := &Request{Payload: &a2a.SendMessageRequest{}}

	ctx, result, err := interceptor.Before(ctx, callCtx, req)
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

	if msg := entry["msg"].(string); msg != "request received" {
		t.Errorf("msg = %q, want %q", msg, "request received")
	}

	_ = ctx
}

func TestLoggingInterceptorAfterSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := log.AttachLogger(t.Context(), logger)

	interceptor := NewLoggingInterceptor()
	callCtx := &CallContext{method: "GetTask"}
	resp := &Response{
		Payload: &a2a.Task{ID: "task-1", Status: a2a.TaskStatus{State: a2a.TaskStateCompleted}},
	}

	if err := interceptor.After(ctx, callCtx, resp); err != nil {
		t.Fatalf("After() error = %v", err)
	}

	if !strings.Contains(buf.String(), "request completed") {
		t.Errorf("log output = %q, want to contain %q", buf.String(), "request completed")
	}
}

func TestLoggingInterceptorAfterClientError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := log.AttachLogger(t.Context(), logger)

	interceptor := NewLoggingInterceptor()
	callCtx := &CallContext{method: "GetTask"}
	resp := &Response{Err: a2a.ErrTaskNotFound}

	if err := interceptor.After(ctx, callCtx, resp); err != nil {
		t.Fatalf("After() error = %v", err)
	}

	if !strings.Contains(buf.String(), "WARN") {
		t.Errorf("client errors should be logged at WARN level, got: %s", buf.String())
	}
}

func TestLoggingInterceptorAfterServerError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := log.AttachLogger(t.Context(), logger)

	interceptor := NewLoggingInterceptor()
	callCtx := &CallContext{method: "SendMessage"}
	resp := &Response{Err: fmt.Errorf("database connection lost")}

	if err := interceptor.After(ctx, callCtx, resp); err != nil {
		t.Fatalf("After() error = %v", err)
	}

	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("server errors should be logged at ERROR level, got: %s", buf.String())
	}
}

func TestLoggingInterceptorPayloadLevel(t *testing.T) {
	t.Run("info logger does not log payload at debug level", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
		ctx := log.AttachLogger(t.Context(), logger)

		interceptor := NewLoggingInterceptor()
		callCtx := &CallContext{method: "SendMessage"}
		req := &Request{Payload: &a2a.SendMessageRequest{}}

		_, _, _ = interceptor.Before(ctx, callCtx, req)

		if strings.Contains(buf.String(), "payload") {
			t.Errorf("payload should not appear at info level: %s", buf.String())
		}
	})

	t.Run("debug logger includes payload", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		ctx := log.AttachLogger(t.Context(), logger)

		interceptor := NewLoggingInterceptor()
		callCtx := &CallContext{method: "SendMessage"}
		req := &Request{Payload: &a2a.SendMessageRequest{}}

		_, _, _ = interceptor.Before(ctx, callCtx, req)

		if !strings.Contains(buf.String(), "payload") {
			t.Errorf("payload should appear at debug level: %s", buf.String())
		}
	})
}
