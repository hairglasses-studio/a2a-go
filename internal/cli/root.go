package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type globalConfig struct {
	output    string
	transport string
	svcParams []string
	auth      string
	tenant    string
	timeout   time.Duration
	verbose   bool
}

func (g *globalConfig) logf(format string, args ...any) {
	if g.verbose {
		fmt.Fprintf(os.Stderr, "# "+format+"\n", args...)
	}
}

// Execute runs the CLI and returns the exit code.
func Execute() int {
	cfg := &globalConfig{}
	root := newRootCmd(cfg)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func newRootCmd(cfg *globalConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "a2a",
		Short:         "CLI for the Agent-to-Agent protocol",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := cmd.PersistentFlags()
	pf.StringVarP(&cfg.output, "output", "o", "text", "Output format: text, json")
	pf.StringVar(&cfg.transport, "transport", "", "Force transport: rest, jsonrpc, grpc")
	pf.StringArrayVar(&cfg.svcParams, "svc-param", nil, "Service parameter k=v (repeatable)")
	pf.StringVar(&cfg.auth, "auth", "", "Shorthand for --svc-param Authorization=<creds>")
	pf.StringVar(&cfg.tenant, "tenant", "", "Tenant identifier")
	pf.DurationVar(&cfg.timeout, "timeout", 30*time.Second, "Request timeout")
	pf.BoolVarP(&cfg.verbose, "verbose", "v", false, "Verbose output to stderr")

	cmd.AddCommand(
		newDiscoverCmd(cfg),
		newSendCmd(cfg),
		newGetCmd(cfg),
		newListCmd(cfg),
		newCancelCmd(cfg),
		newSubscribeCmd(cfg),
		newServeCmd(cfg),
	)

	return cmd
}
