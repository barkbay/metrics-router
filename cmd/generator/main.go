package generator

import "github.com/spf13/cobra"

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate resources according to the current cluster configuration",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO
		},
	}
	return cmd
}
