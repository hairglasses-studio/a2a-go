package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func newGetCmd(cfg *globalConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a resource",
	}
	cmd.AddCommand(newGetTaskCmd(cfg))
	return cmd
}

func newGetTaskCmd(cfg *globalConfig) *cobra.Command {
	var history int

	cmd := &cobra.Command{
		Use:   "task <url> <task-id>",
		Short: "Get task details",
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

			req := &a2a.GetTaskRequest{
				ID:     a2a.TaskID(args[1]),
				Tenant: cfg.tenant,
			}
			if cmd.Flags().Changed("history") {
				req.HistoryLength = &history
			}

			task, err := client.GetTask(ctx, req)
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

	cmd.Flags().IntVar(&history, "history", 0, "Include up to n history messages")
	return cmd
}
