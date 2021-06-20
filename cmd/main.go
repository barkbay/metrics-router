package main

import (
	"github.com/barkbay/custom-metrics-router/cmd/generator"
	"github.com/barkbay/custom-metrics-router/cmd/server"
	"github.com/spf13/cobra"
)

func main() {

	rootCmd := &cobra.Command{
		Use:          "metrics-router",
		Short:        "Kubernetes metrics router",
		Version:      "0.0.1",
		SilenceUsage: true,
	}
	rootCmd.AddCommand(server.Command(), generator.Command())
	_ = rootCmd.Execute()
}
