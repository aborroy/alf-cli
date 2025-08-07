package alfresco

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var TemplateFS embed.FS

var rootCmd = &cobra.Command{
	Use:   "alfresco",
	Short: "alfresco - The Alfresco CLI",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() {
	fmt.Println("Welcome to the Alfresco CLI!\nUse 'alfresco --help' to see available commands.")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
