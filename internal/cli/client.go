package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
)

func newClient(ctx context.Context, cfg *globalConfig, agentURL string) (*a2aclient.Client, error) {
	cfg.logf("resolving agent card from %s", agentURL)

	var resolveOpts []agentcard.ResolveOption
	if cfg.auth != "" {
		resolveOpts = append(resolveOpts, agentcard.WithRequestHeader("Authorization", cfg.auth))
	}
	card, err := agentcard.DefaultResolver.Resolve(ctx, agentURL, resolveOpts...)
	if err != nil {
		return nil, fmt.Errorf("resolving agent card: %w", err)
	}

	var factoryOpts []a2aclient.FactoryOption
	if cfg.transport != "" {
		proto, err := parseTransport(cfg.transport)
		if err != nil {
			return nil, err
		}
		factoryOpts = append(factoryOpts, a2aclient.WithConfig(a2aclient.Config{
			PreferredTransports: []a2a.TransportProtocol{proto},
		}))
	}

	cfg.logf("creating client for %s", card.Name)
	return a2aclient.NewFromCard(ctx, card, factoryOpts...)
}

func withServiceParams(ctx context.Context, cfg *globalConfig) context.Context {
	params := a2aclient.ServiceParams{}
	for _, kv := range cfg.svcParams {
		k, v, _ := strings.Cut(kv, "=")
		params.Append(k, v)
	}
	if cfg.auth != "" {
		params.Append("Authorization", cfg.auth)
	}
	if len(params) > 0 {
		ctx = a2aclient.AttachServiceParams(ctx, params)
	}
	return ctx
}

func parseTransport(s string) (a2a.TransportProtocol, error) {
	switch strings.ToLower(s) {
	case "rest":
		return a2a.TransportProtocolHTTPJSON, nil
	case "jsonrpc":
		return a2a.TransportProtocolJSONRPC, nil
	case "grpc":
		return a2a.TransportProtocolGRPC, nil
	default:
		return "", fmt.Errorf("unknown transport %q (use rest, jsonrpc, or grpc)", s)
	}
}
