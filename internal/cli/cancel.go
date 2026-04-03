package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func newCancelCmd(cfg *globalConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <url> <task-id>",
		Short: "Cancel a task",
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

			task, err := client.CancelTask(ctx, &a2a.CancelTaskRequest{
				ID:     a2a.TaskID(args[1]),
				Tenant: cfg.tenant,
			})
			if err != nil {
				return err
			}

			if cfg.output == "json" {
				return printJSON(os.Stdout, task)
			}
			printTask(os.Stdout, task)
			return nil
		},
	}
}
