package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

func newServeCmd(cfg *globalConfig) *cobra.Command {
	var (
		port     int
		host     string
		name     string
		desc     string
		cardFile string
		quiet    bool
		echo     bool
		proxyURL string
		execCmd  string
		chunk    string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start an A2A-compliant server",
		RunE: func(cmd *cobra.Command, args []string) error {
			modes := 0
			if echo {
				modes++
			}
			if proxyURL != "" {
				modes++
			}
			if execCmd != "" {
				modes++
			}
			if modes > 1 {
				return fmt.Errorf("--echo, --proxy, and --exec are mutually exclusive")
			}
			if modes == 0 {
				return fmt.Errorf("specify --echo, --proxy <url>, or --exec <cmd>")
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
			if err != nil {
				return fmt.Errorf("listen: %w", err)
			}

			addr := listener.Addr().String()
			serveTransport := cfg.transport
			if serveTransport == "" {
				serveTransport = "rest"
			}

			proto := a2a.TransportProtocolHTTPJSON
			if serveTransport == "jsonrpc" {
				proto = a2a.TransportProtocolJSONRPC
			}

			switch {
			case echo:
				return serveEcho(ctx, cfg, listener, addr, proto, name, desc, cardFile, quiet)
			case proxyURL != "":
				return serveProxy(ctx, cfg, listener, addr, proto, proxyURL, cardFile, quiet)
			default:
				return serveExec(ctx, cfg, listener, addr, proto, execCmd, chunk, name, desc, cardFile, quiet)
			}
		},
	}

	f := cmd.Flags()
	f.IntVar(&port, "port", 8080, "Listen port")
	f.StringVar(&host, "host", "127.0.0.1", "Bind address")
	f.StringVar(&name, "name", "", "Agent name for the auto-generated card")
	f.StringVar(&desc, "description", "", "Agent description")
	f.StringVar(&cardFile, "card", "", "Serve a custom agent card JSON file")
	f.BoolVar(&quiet, "quiet", false, "Suppress traffic logging to stderr")
	f.BoolVar(&echo, "echo", false, "Echo mode: return the user's message as a response")
	f.StringVar(&proxyURL, "proxy", "", "Proxy mode: forward requests to an upstream agent URL")
	f.StringVar(&execCmd, "exec", "", "Exec mode: run a command as an A2A agent")
	f.StringVar(&chunk, "chunk", "", "Delimiter for streaming exec output (implies --exec)")

	return cmd
}

func loadOrBuildCard(cardFile, name, desc, addr string, proto a2a.TransportProtocol) (*a2a.AgentCard, error) {
	if cardFile != "" {
		data, err := os.ReadFile(cardFile)
		if err != nil {
			return nil, fmt.Errorf("reading card file: %w", err)
		}
		card := new(a2a.AgentCard)
		if err := json.Unmarshal(data, card); err != nil {
			return nil, fmt.Errorf("parsing card file: %w", err)
		}
		return card, nil
	}

	if name == "" {
		name = "a2a-cli"
	}
	url := "http://" + addr
	return &a2a.AgentCard{
		Name:        name,
		Description: desc,
		Version:     "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(url, proto),
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}, nil
}

func startServer(ctx context.Context, listener net.Listener, handler http.Handler, quiet bool) error {
	addr := listener.Addr().String()
	srv := &http.Server{Handler: handler}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	if !quiet {
		fmt.Fprintf(os.Stderr, "Listening on %s\n", addr)
	}

	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func buildMux(handler a2asrv.RequestHandler, card *a2a.AgentCard, transport string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))

	switch transport {
	case "jsonrpc":
		mux.Handle("/", a2asrv.NewJSONRPCHandler(handler))
	default:
		mux.Handle("/", a2asrv.NewRESTHandler(handler))
	}
	return mux
}

func serveEcho(ctx context.Context, cfg *globalConfig, listener net.Listener, addr string, proto a2a.TransportProtocol, name, desc, cardFile string, quiet bool) error {
	if name == "" {
		name = "Echo Agent"
	}
	if desc == "" {
		desc = "Echoes the user's message back as a response"
	}

	card, err := loadOrBuildCard(cardFile, name, desc, addr, proto)
	if err != nil {
		return err
	}

	handler := a2asrv.NewHandler(&echoExecutor{},
		a2asrv.WithCapabilityChecks(&a2a.AgentCapabilities{Streaming: true}),
	)
	transport := cfg.transport
	if transport == "" {
		transport = "rest"
	}
	mux := buildMux(handler, card, transport)

	cfg.logf("echo mode, transport=%s", transport)
	return startServer(ctx, listener, mux, quiet)
}

type echoExecutor struct{}

func (e *echoExecutor) Execute(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}
		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}
		text := messageText(execCtx.Message)
		evt := a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(text))
		evt.LastChunk = true
		if !yield(evt, nil) {
			return
		}
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
	}
}

func (e *echoExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}
