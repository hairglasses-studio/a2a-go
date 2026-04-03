package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"iter"
	"net"
	"os/exec"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

func serveExec(ctx context.Context, cfg *globalConfig, listener net.Listener, addr string, proto a2a.TransportProtocol, command, chunk, name, desc, cardFile string, quiet bool) error {
	if name == "" {
		name = "Exec Agent"
	}
	if desc == "" {
		desc = fmt.Sprintf("Wraps command: %s", command)
	}

	card, err := loadOrBuildCard(cardFile, name, desc, addr, proto)
	if err != nil {
		return err
	}

	caps := &a2a.AgentCapabilities{Streaming: chunk != ""}
	handler := a2asrv.NewHandler(
		&execExecutor{command: command, chunk: chunk},
		a2asrv.WithCapabilityChecks(caps),
	)
	transport := cfg.transport
	if transport == "" {
		transport = "rest"
	}
	mux := buildMux(handler, card, transport)

	cfg.logf("exec mode, command=%q chunk=%q transport=%s", command, chunk, transport)
	return startServer(ctx, listener, mux, quiet)
}

type execExecutor struct {
	command string
	chunk   string
}

func (e *execExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}
		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}
		input := messageText(execCtx.Message)

		if e.chunk == "" {
			e.executeBuffered(ctx, execCtx, input, yield)
		} else {
			e.executeChunked(ctx, execCtx, input, yield)
		}
	}
}

func (e *execExecutor) executeBuffered(ctx context.Context, execCtx *a2asrv.ExecutorContext, input string, yield func(a2a.Event, error) bool) {
	cmd := exec.CommandContext(ctx, "sh", "-c", e.command)
	cmd.Stdin = strings.NewReader(input)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed,
			a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(msg))), nil)
		return
	}

	evt := a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(string(output)))
	evt.LastChunk = true
	if !yield(evt, nil) {
		return
	}
	yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
}

func (e *execExecutor) executeChunked(ctx context.Context, execCtx *a2asrv.ExecutorContext, input string, yield func(a2a.Event, error) bool) {
	cmd := exec.CommandContext(ctx, "sh", "-c", e.command)
	cmd.Stdin = strings.NewReader(input)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		yield(nil, fmt.Errorf("creating stdout pipe: %w", err))
		return
	}
	if err := cmd.Start(); err != nil {
		yield(nil, fmt.Errorf("starting command: %w", err))
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Split(splitByDelimiter(e.chunk))

	// Buffer one chunk ahead so we can set LastChunk on the final one.
	var pending *string
	var artifactID a2a.ArtifactID

	for scanner.Scan() {
		text := scanner.Text()
		if pending != nil {
			if artifactID == "" {
				evt := a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(*pending))
				artifactID = evt.Artifact.ID
				if !yield(evt, nil) {
					return
				}
			} else {
				evt := a2a.NewArtifactUpdateEvent(execCtx, artifactID, a2a.NewTextPart(*pending))
				evt.Append = true
				if !yield(evt, nil) {
					return
				}
			}
		}
		pending = &text
	}

	// Emit the final chunk with LastChunk=true.
	if pending != nil {
		if artifactID == "" {
			evt := a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(*pending))
			evt.LastChunk = true
			if !yield(evt, nil) {
				return
			}
		} else {
			evt := a2a.NewArtifactUpdateEvent(execCtx, artifactID, a2a.NewTextPart(*pending))
			evt.Append = true
			evt.LastChunk = true
			if !yield(evt, nil) {
				return
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed,
			a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(msg))), nil)
		return
	}
	yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
}

func (e *execExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

func splitByDelimiter(delim string) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, []byte(delim)); i >= 0 {
			return i + len(delim), data[:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
}
