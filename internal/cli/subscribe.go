package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func newSubscribeCmd(cfg *globalConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "subscribe <url> <task-id>",
		Short: "Subscribe to task events",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.timeout)
			defer cancel()
			ctx = withServiceParams(ctx, cfg)

			client, err := newClient(ctx, cfg, args[0])
			if err != nil {
				return err
			}
			defer func() { _ = client.Destroy() }()

			cfg.logf("subscribing to task %s", args[1])

			for event, err := range client.SubscribeToTask(ctx, &a2a.SubscribeToTaskRequest{
				ID:     a2a.TaskID(args[1]),
				Tenant: cfg.tenant,
			}) {
				if err != nil {
					return err
				}
				if cfg.output == "json" {
					if err := printJSON(os.Stdout, event); err != nil {
						return err
					}
				} else {
					printEvent(os.Stdout, event)
				}
			}
			return nil
		},
	}
}
