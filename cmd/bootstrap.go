/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Initialize the bin-manager",
	Long: `Create as a folder for all the binaries managed by bin-manager.
	It will download the binaries yaml from the provided git repo. In case you 
	dit not yet setup the git repo, you can do so by within this command.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := os.Mkdir("bin", 0755)
		if err != nil {
			fmt.Println("Error creating bin directory:", err)
			return
		}
		fmt.Println("bootstrap called")
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bootstrapCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bootstrapCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
