package cli

import (
	"context"
	"fmt"
	"iter"
	"net"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

func serveProxy(ctx context.Context, cfg *globalConfig, listener net.Listener, addr string, proto a2a.TransportProtocol, upstreamURL, cardFile string, quiet bool) error {
	cfg.logf("resolving upstream card from %s", upstreamURL)

	var resolveOpts []agentcard.ResolveOption
	if cfg.auth != "" {
		resolveOpts = append(resolveOpts, agentcard.WithRequestHeader("Authorization", cfg.auth))
	}
	upstreamCard, err := agentcard.DefaultResolver.Resolve(ctx, upstreamURL, resolveOpts...)
	if err != nil {
		return fmt.Errorf("resolving upstream agent card: %w", err)
	}

	client, err := a2aclient.NewFromCard(ctx, upstreamCard)
	if err != nil {
		return fmt.Errorf("creating upstream client: %w", err)
	}

	svcParams := buildServiceParams(cfg)

	var card *a2a.AgentCard
	if cardFile != "" {
		card, err = loadOrBuildCard(cardFile, "", "", addr, proto)
		if err != nil {
			return err
		}
	} else {
		card = deriveProxyCard(upstreamCard, addr, proto)
	}

	handler := &proxyHandler{client: client, svcParams: svcParams}
	transport := cfg.transport
	if transport == "" {
		transport = "rest"
	}
	mux := buildMux(handler, card, transport)

	if !quiet {
		fmt.Fprintf(os.Stderr, "Proxying to %s\n", upstreamURL)
	}
	cfg.logf("proxy mode, transport=%s", transport)
	return startServer(ctx, listener, mux, quiet)
}

func deriveProxyCard(upstream *a2a.AgentCard, addr string, proto a2a.TransportProtocol) *a2a.AgentCard {
	card := *upstream
	card.SupportedInterfaces = []*a2a.AgentInterface{
		a2a.NewAgentInterface("http://"+addr, proto),
	}
	return &card
}

func buildServiceParams(cfg *globalConfig) a2aclient.ServiceParams {
	params := a2aclient.ServiceParams{}
	for _, kv := range cfg.svcParams {
		k, v, _ := strings.Cut(kv, "=")
		params.Append(k, v)
	}
	if cfg.auth != "" {
		params.Append("Authorization", cfg.auth)
	}
	return params
}

type proxyHandler struct {
	client    *a2aclient.Client
	svcParams a2aclient.ServiceParams
}

func (p *proxyHandler) withParams(ctx context.Context) context.Context {
	if len(p.svcParams) > 0 {
		return a2aclient.AttachServiceParams(ctx, p.svcParams)
	}
	return ctx
}

func (p *proxyHandler) GetTask(ctx context.Context, req *a2a.GetTaskRequest) (*a2a.Task, error) {
	return p.client.GetTask(p.withParams(ctx), req)
}

func (p *proxyHandler) ListTasks(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	return p.client.ListTasks(p.withParams(ctx), req)
}

func (p *proxyHandler) CancelTask(ctx context.Context, req *a2a.CancelTaskRequest) (*a2a.Task, error) {
	return p.client.CancelTask(p.withParams(ctx), req)
}

func (p *proxyHandler) SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error) {
	return p.client.SendMessage(p.withParams(ctx), req)
}

func (p *proxyHandler) SubscribeToTask(ctx context.Context, req *a2a.SubscribeToTaskRequest) iter.Seq2[a2a.Event, error] {
	return p.client.SubscribeToTask(p.withParams(ctx), req)
}

func (p *proxyHandler) SendStreamingMessage(ctx context.Context, req *a2a.SendMessageRequest) iter.Seq2[a2a.Event, error] {
	return p.client.SendStreamingMessage(p.withParams(ctx), req)
}

func (p *proxyHandler) GetTaskPushConfig(ctx context.Context, req *a2a.GetTaskPushConfigRequest) (*a2a.TaskPushConfig, error) {
	return p.client.GetTaskPushConfig(p.withParams(ctx), req)
}

func (p *proxyHandler) ListTaskPushConfigs(ctx context.Context, req *a2a.ListTaskPushConfigRequest) ([]*a2a.TaskPushConfig, error) {
	return p.client.ListTaskPushConfigs(p.withParams(ctx), req)
}

func (p *proxyHandler) CreateTaskPushConfig(ctx context.Context, req *a2a.CreateTaskPushConfigRequest) (*a2a.TaskPushConfig, error) {
	return p.client.CreateTaskPushConfig(p.withParams(ctx), req)
}

func (p *proxyHandler) DeleteTaskPushConfig(ctx context.Context, req *a2a.DeleteTaskPushConfigRequest) error {
	return p.client.DeleteTaskPushConfig(p.withParams(ctx), req)
}

func (p *proxyHandler) GetExtendedAgentCard(ctx context.Context, req *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error) {
	return p.client.GetExtendedAgentCard(p.withParams(ctx), req)
}

var _ a2asrv.RequestHandler = (*proxyHandler)(nil)
