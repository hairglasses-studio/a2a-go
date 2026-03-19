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

// Package main demonstrates how to configure observability for an A2A server.
//
// It shows how to:
//   - Set up structured logging with configurable log levels (debug, info, warn)
//   - Redirect logs to a file using the -logfile flag
//   - Use the built-in [a2asrv.LoggingInterceptor] for request/response logging
//   - Pass a custom logger to the request handler via [a2asrv.WithLogger]
//
// Usage:
//
//	# Default: info level, logs to stderr
//	go run ./examples/observability/server
//
//	# Debug level to see request/response payloads
//	go run ./examples/observability/server -loglevel debug
//
//	# Write logs to a file
//	go run ./examples/observability/server -logfile server.log
//
//	# Both: debug level written to a file
//	go run ./examples/observability/server -loglevel debug -logfile server.log
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"iter"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

type agentExecutor struct{}

var _ a2asrv.AgentExecutor = (*agentExecutor)(nil)

func (*agentExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}

		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}

		time.Sleep(100 * time.Millisecond)

		msg := execCtx.Message
		var inputText string
		for _, part := range msg.Parts {
			inputText = part.Text()
			if inputText != "" {
				break
			}
		}

		if strings.Contains(strings.ToLower(inputText), "fail") {
			failMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("Simulated failure"))
			yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed, failMsg), nil)
			return
		}

		if !yield(a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(fmt.Sprintf("Echo: %s", inputText))), nil) {
			return
		}

		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
	}
}

func (*agentExecutor) Cancel(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

var (
	port     = flag.Int("port", 9001, "Port for the server to listen on.")
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

	// Set the slog default so that any code using slog.Default() also gets this logger.
	slog.SetDefault(logger)

	addr := fmt.Sprintf("http://127.0.0.1:%d/invoke", *port)
	agentCard := &a2a.AgentCard{
		Name:        "Observability Demo Agent",
		Description: "An agent demonstrating observability setup",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(addr, a2a.TransportProtocolJSONRPC),
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
		Skills: []a2a.AgentSkill{{
			ID:          "echo",
			Name:        "Echo",
			Description: "Echoes input back. Send 'fail' to trigger an error.",
		}},
	}

	requestHandler := a2asrv.NewHandler(
		&agentExecutor{},
		// Attach a configured logger: all request-scoped log entries will inherit its handler.
		a2asrv.WithLogger(logger),
		// Add the logging interceptor for request/response lifecycle visibility.
		a2asrv.WithCallInterceptors(a2asrv.NewLoggingInterceptor()),
	)

	mux := http.NewServeMux()
	mux.Handle("/invoke", a2asrv.NewJSONRPCHandler(requestHandler))
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to bind to a port: %v", err)
	}
	logger.Info("server starting", "port", *port, "loglevel", *logLevel)

	if err := http.Serve(listener, mux); !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped unexpectedly", "error", err)
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
