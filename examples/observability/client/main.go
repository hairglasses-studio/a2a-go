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

// Package main demonstrates how to configure observability for an A2A client.
//
// It shows how to:
//   - Set up structured logging with configurable log levels
//   - Use the built-in [a2aclient.LoggingInterceptor] for request/response logging
//   - Attach a logger to the context so the interceptor picks it up
//   - Redirect logs to a file
//
// Usage:
//
//	# Default: info level, logs to stderr
//	go run ./examples/observability/client
//
//	# Debug level to see request/response payloads
//	go run ./examples/observability/client -loglevel debug
//
//	# Write logs to a file
//	go run ./examples/observability/client -logfile client.log
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2alog "github.com/a2aproject/a2a-go/v2/log"
)

var (
	cardURL  = flag.String("card-url", "http://127.0.0.1:9001", "Base URL of AgentCard server.")
	logLevel = flag.String("loglevel", "info", "Log level: debug, info, warn, error.")
	logFile  = flag.String("logfile", "", "Path to log file. If empty, logs go to stderr.")
)

func main() {
	flag.Parse()

	logger, cleanup, err := newLogger(*logLevel, *logFile)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer cleanup()

	slog.SetDefault(logger)

	// Attach the logger to the context. The logging interceptor and transport-level
	// code will use log.LoggerFrom(ctx) to retrieve it.
	ctx := a2alog.AttachLogger(context.Background(), logger)

	card, err := agentcard.DefaultResolver.Resolve(ctx, *cardURL)
	if err != nil {
		log.Fatalf("Failed to resolve AgentCard: %v", err)
	}

	client, err := a2aclient.NewFromCard(ctx, card,
		// Add the logging interceptor for request/response lifecycle visibility.
		a2aclient.WithCallInterceptors(a2aclient.NewLoggingInterceptor()),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("Hello from the observability example"))
	resp, err := client.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})
	if err != nil {
		logger.Error("SendMessage failed", "error", err)
		return
	}
	logger.Info("SendMessage response", "result", fmt.Sprintf("%+v", resp))

	// Demonstrate streaming call visibility.
	streamMsg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("Streaming hello"))
	for event, err := range client.SendStreamingMessage(ctx, &a2a.SendMessageRequest{Message: streamMsg}) {
		if err != nil {
			logger.Error("streaming event error", "error", err)
			break
		}
		logger.Info("streaming event received", "event_type", fmt.Sprintf("%T", event))
	}
}

func newLogger(level, file string) (*slog.Logger, func(), error) {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		return nil, nil, fmt.Errorf("unsupported log level: %q (use debug, info, warn, error)", level)
	}

	opts := &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: slogLevel <= slog.LevelDebug,
	}

	cleanup := func() {}
	writer := os.Stderr

	if file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writer = f
		cleanup = func() { f.Close() }
	}

	handler := slog.NewJSONHandler(writer, opts)
	return slog.New(handler), cleanup, nil
}
