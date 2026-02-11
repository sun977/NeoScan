package main

import (
	"fmt"

	"neoagent/internal/pkg/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  "显示 NeoScan-Agent 的版本信息，包括版本号、构建时间、Git 提交和 Go 版本。",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("NeoScan-Agent %s\n", version.GetVersion())
		fmt.Printf("API Version: %s\n", version.APIVersion)
		fmt.Printf("Build Time: %s\n", version.BuildTime)
		fmt.Printf("Git Commit: %s\n", version.GitCommit)
		fmt.Printf("Go Version: %s\n", version.GoVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
