/*
Copyright © 2025 acejilam acejilam@gmail.com
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var progVersion string
var progCommit string
var progStatus string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "version",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", progVersion)
		fmt.Println("Commit:", progCommit)
		fmt.Println("Status:", progStatus)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
