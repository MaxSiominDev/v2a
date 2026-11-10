package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MaxSiominDev/v2a/internal/application"
	"github.com/MaxSiominDev/v2a/internal/buildinfo"
	"github.com/MaxSiominDev/v2a/internal/media"
)

var verbose bool

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "v2a",
		Short:         "Extract and compress audio from media files",
		Long:          "v2a converts MOV audio to MP3, compresses existing MP3s and inspects media files.\nIt requires ffmpeg and ffprobe to be installed.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildinfo.Version,
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show ffmpeg/ffprobe diagnostic output")

	root.AddCommand(
		newExtractCmd(),
		newCompressCmd(),
		newInfoCmd(),
		newVersionCmd(),
	)
	return root
}

func Execute() int {
	err := NewRootCmd().Execute()
	if err != nil {
		printError(err)
		return 1
	}
	return 0
}

// printError keeps normal errors short; raw tool output is verbose-only.
func printError(err error) {
	var enc *application.EncodeError
	if errors.As(err, &enc) {
		fmt.Fprintln(os.Stderr, "error: encoding failed")
		printToolStderr(enc.Stderr)
		return
	}

	var probeErr *media.ProbeError
	if errors.As(err, &probeErr) {
		fmt.Fprintln(os.Stderr, "error:", probeErr)
		printToolStderr(probeErr.Stderr)
		return
	}

	fmt.Fprintln(os.Stderr, "error:", err)
}

func printToolStderr(stderr string) {
	if verbose {
		fmt.Fprintln(os.Stderr, indentBlock(stderr))
		return
	}
	fmt.Fprintln(os.Stderr, "re-run with --verbose to see the tool output")
}
