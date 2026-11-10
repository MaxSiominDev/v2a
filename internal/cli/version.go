package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MaxSiominDev/v2a/internal/buildinfo"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("v2a version %s\n", buildinfo.Version)
			return nil
		},
	}
}
