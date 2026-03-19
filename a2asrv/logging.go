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
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/log"
)

// LoggingInterceptor is a [CallInterceptor] that logs incoming requests and outgoing responses.
// It uses the context-scoped logger attached by [InterceptedHandler] (which already carries
// request_id, task_id, and other A2A attributes).
//
// At Info level (the default), it logs method name and call duration.
// At Debug level, it additionally logs request/response payloads and error details.
// Errors in responses are always logged at Warn level.
//
// Use [NewLoggingInterceptor] to create an instance with default settings.
type LoggingInterceptor struct {
	// PayloadLogLevel controls the minimum level at which request and response payloads are
	// included in log output. Defaults to [slog.LevelDebug].
	PayloadLogLevel slog.Level
}

type loggingStartTimeKey struct{}

var _ CallInterceptor = (*LoggingInterceptor)(nil)

// NewLoggingInterceptor creates a [LoggingInterceptor] with default settings.
func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{PayloadLogLevel: slog.LevelDebug}
}

// Before implements [CallInterceptor].
func (l *LoggingInterceptor) Before(ctx context.Context, callCtx *CallContext, req *Request) (context.Context, any, error) {
	ctx = context.WithValue(ctx, loggingStartTimeKey{}, time.Now())
	logger := log.LoggerFrom(ctx)
	if logger.Enabled(ctx, l.PayloadLogLevel) {
		log.Log(ctx, l.PayloadLogLevel, "request received", "method", callCtx.Method(), "payload", req.Payload)
	} else {
		log.Info(ctx, "request received", "method", callCtx.Method())
	}
	return ctx, nil, nil
}

// After implements [CallInterceptor].
func (l *LoggingInterceptor) After(ctx context.Context, callCtx *CallContext, resp *Response) error {
	duration := time.Duration(0)
	if start, ok := ctx.Value(loggingStartTimeKey{}).(time.Time); ok {
		duration = time.Since(start)
	}

	if resp.Err != nil {
		attrs := []any{
			"method", callCtx.Method(),
			"duration", duration,
			"error", resp.Err,
		}
		if isClientError(resp.Err) {
			log.Warn(ctx, "request failed", attrs...)
		} else {
			log.Error(ctx, "request failed", resp.Err, attrs[0:4]...)
		}
		return nil
	}

	logger := log.LoggerFrom(ctx)
	if logger.Enabled(ctx, l.PayloadLogLevel) {
		log.Log(ctx, l.PayloadLogLevel, "request completed", "method", callCtx.Method(), "duration", duration, "payload", formatResponsePayload(resp.Payload))
	} else {
		log.Info(ctx, "request completed", "method", callCtx.Method(), "duration", duration)
	}
	return nil
}

func isClientError(err error) bool {
	return errors.Is(err, a2a.ErrInvalidParams) ||
		errors.Is(err, a2a.ErrInvalidRequest) ||
		errors.Is(err, a2a.ErrTaskNotFound) ||
		errors.Is(err, a2a.ErrParseError) ||
		errors.Is(err, a2a.ErrMethodNotFound) ||
		errors.Is(err, a2a.ErrUnauthenticated)
}

func formatResponsePayload(payload any) any {
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
