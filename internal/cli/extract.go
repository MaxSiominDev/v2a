package cli

import (
	"github.com/spf13/cobra"

	"github.com/MaxSiominDev/v2a/internal/application"
	"github.com/MaxSiominDev/v2a/internal/compress"
)

func newExtractCmd() *cobra.Command {
	var (
		bitrate string
		output  string
	)

	cmd := &cobra.Command{
		Use:   "extract <input>",
		Short: "Extract the audio of a MOV or MP4 file to MP3",
		Args:  cobra.ExactArgs(1),
		Example: `  v2a extract movie.mov
  v2a extract movie.mov --bitrate 192k
  v2a extract movie.mov -o ~/Music/movie.mp3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := application.ExtractRequest{Input: args[0], Output: output}

			if bitrate != "" {
				bps, err := compress.ParseBitrate(bitrate)
				if err != nil {
					return err
				}
				req.Bitrate = bps
			}

			if err := application.ValidateExtract(req); err != nil {
				return err
			}

			app, err := application.New()
			if err != nil {
				return err
			}

			header("Extracting audio...")
			res, err := app.Extract(cmd.Context(), req)
			if err != nil {
				return err
			}

			report(req.Input, res.OutputPath, res.Notes)
			return nil
		},
	}

	cmd.Flags().StringVar(&bitrate, "bitrate", "", "encode at this bitrate, e.g. 192k; by default a high-quality VBR preset is used")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to this exact path instead of the default name")

	return cmd
}
