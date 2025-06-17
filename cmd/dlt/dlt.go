package dlt

import (
	"log/slog"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dlt",
		Short: "DLT CLI",
		Long:  "DLT CLI (stub, to be filled in)",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize cmdio context
			cmdIO := cmdio.NewIO(cmd.Context(), flags.OutputText, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", "")
			ctx := cmdio.InContext(cmd.Context(), cmdIO)

			// Set up logger with WARN level
			h := handler.NewFriendlyHandler(cmd.ErrOrStderr(), &handler.Options{
				Color: cmdio.IsTTY(cmd.ErrOrStderr()),
				Level: log.LevelWarn,
			})
			logger := slog.New(h)
			ctx = log.NewContext(ctx, logger)

			cmd.SetContext(ctx)
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(initCommand())

	return cmd
}
