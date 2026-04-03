package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
)

func newDiscoverCmd(cfg *globalConfig) *cobra.Command {
	var extended bool

	cmd := &cobra.Command{
		Use:   "discover <url>",
		Short: "Fetch and display an agent card",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiscover(cmd.Context(), cfg, args[0], extended)
		},
	}

	cmd.Flags().BoolVar(&extended, "extended", false, "Fetch the extended agent card")
	return cmd
}

func runDiscover(ctx context.Context, cfg *globalConfig, agentURL string, extended bool) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	if extended {
		ctx = withServiceParams(ctx, cfg)
		client, err := newClient(ctx, cfg, agentURL)
		if err != nil {
			return err
		}
		defer func() { _ = client.Destroy() }()

		card, err := client.GetExtendedAgentCard(ctx, &a2a.GetExtendedAgentCardRequest{
			Tenant: cfg.tenant,
		})
		if err != nil {
			return err
		}
		if cfg.output == "json" {
			return printJSON(os.Stdout, card)
		}
		printCard(os.Stdout, card)
		return nil
	}

	var resolveOpts []agentcard.ResolveOption
	if cfg.auth != "" {
		resolveOpts = append(resolveOpts, agentcard.WithRequestHeader("Authorization", cfg.auth))
	}
	cfg.logf("fetching agent card from %s", agentURL)
	card, err := agentcard.DefaultResolver.Resolve(ctx, agentURL, resolveOpts...)
	if err != nil {
		return err
	}

	if cfg.output == "json" {
		return printJSON(os.Stdout, card)
	}
	printCard(os.Stdout, card)
	return nil
}
