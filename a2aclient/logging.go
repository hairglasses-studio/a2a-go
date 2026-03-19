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
	"context"
	"log/slog"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/log"
)

// LoggingInterceptor is a [CallInterceptor] that logs outgoing requests and incoming responses.
//
// At Info level (the default), it logs method name, target URL, and call duration.
// At Debug level, it additionally logs request/response payloads.
// Errors in responses are always logged at Warn level.
//
// The interceptor uses the context-scoped logger from the [log] package. Callers can attach
// a configured [slog.Logger] to the context using [log.AttachLogger] before making client calls.
//
// Use [NewLoggingInterceptor] to create an instance with default settings.
type LoggingInterceptor struct {
	// PayloadLogLevel controls the minimum level at which request and response payloads are
	// included in log output. Defaults to [slog.LevelDebug].
	PayloadLogLevel slog.Level
}

type clientLoggingStartTimeKey struct{}

var _ CallInterceptor = (*LoggingInterceptor)(nil)

// NewLoggingInterceptor creates a [LoggingInterceptor] with default settings.
func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{PayloadLogLevel: slog.LevelDebug}
}

// Before implements [CallInterceptor].
func (l *LoggingInterceptor) Before(ctx context.Context, req *Request) (context.Context, any, error) {
	ctx = context.WithValue(ctx, clientLoggingStartTimeKey{}, time.Now())
	logger := log.LoggerFrom(ctx)
	if logger.Enabled(ctx, l.PayloadLogLevel) {
		log.Log(ctx, l.PayloadLogLevel, "sending request", "method", req.Method, "url", req.BaseURL, "payload", req.Payload)
	} else {
		log.Info(ctx, "sending request", "method", req.Method, "url", req.BaseURL)
	}
	return ctx, nil, nil
}

// After implements [CallInterceptor].
func (l *LoggingInterceptor) After(ctx context.Context, resp *Response) error {
	duration := time.Duration(0)
	if start, ok := ctx.Value(clientLoggingStartTimeKey{}).(time.Time); ok {
		duration = time.Since(start)
	}

	if resp.Err != nil {
		log.Warn(ctx, "request failed", "method", resp.Method, "url", resp.BaseURL, "duration", duration, "error", resp.Err)
		return nil
	}

	logger := log.LoggerFrom(ctx)
	if logger.Enabled(ctx, l.PayloadLogLevel) {
		log.Log(ctx, l.PayloadLogLevel, "response received", "method", resp.Method, "url", resp.BaseURL, "duration", duration, "payload", formatClientResponsePayload(resp.Payload))
	} else {
		log.Info(ctx, "response received", "method", resp.Method, "url", resp.BaseURL, "duration", duration)
	}
	return nil
}

func formatClientResponsePayload(payload any) any {
	switch v := payload.(type) {
	case *a2a.Task:
		if v == nil {
			return nil
		}
		return slog.GroupValue(
			slog.String("task_id", string(v.ID)),
			slog.String("state", string(v.Status.State)),
		)
	case *a2a.Message:
		if v == nil {
			return nil
		}
		return slog.GroupValue(
			slog.String("message_id", v.ID),
			slog.String("role", string(v.Role)),
			slog.Int("parts", len(v.Parts)),
		)
	default:
		return payload
	}
}
