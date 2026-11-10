package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MaxSiominDev/v2a/internal/application"
	"github.com/MaxSiominDev/v2a/internal/compress"
)

func newCompressCmd() *cobra.Command {
	var (
		size    string
		bitrate string
		output  string
	)

	cmd := &cobra.Command{
		Use:   "compress <input>",
		Short: "Re-encode an MP3 to make it smaller (lossy)",
		Long:  "Compress re-encodes an MP3 file to reduce its size. This is a lossy operation:\nquality is lost even at the same bitrate. The input file is never modified.",
		Args:  cobra.ExactArgs(1),
		Example: `  v2a compress song.mp3 --size 80mb
  v2a compress song.mp3 --bitrate 128k
  v2a compress song.mp3 --size 80mb -o smaller.mp3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if size != "" && bitrate != "" {
				return fmt.Errorf("options --size and --bitrate are mutually exclusive")
			}
			if size == "" && bitrate == "" {
				return fmt.Errorf("one of --size or --bitrate is required")
			}

			req := application.CompressRequest{Input: args[0], Output: output}
			if size != "" {
				bytes, err := compress.ParseSize(size)
				if err != nil {
					return err
				}
				req.TargetBytes = bytes
			} else {
				bps, err := compress.ParseBitrate(bitrate)
				if err != nil {
					return err
				}
				req.Bitrate = bps
			}

			if err := application.ValidateCompress(req); err != nil {
				return err
			}

			app, err := application.New()
			if err != nil {
				return err
			}

			header("Compressing...")
			res, err := app.Compress(cmd.Context(), req)
			if err != nil {
				return err
			}

			if res.AlreadyWithinTarget {
				if res.Copied {
					report(req.Input, res.OutputPath, res.Notes)
				} else {
					fmt.Printf("Input:  %s\n", req.Input)
					fmt.Printf("Input is already within the target size; nothing to do.\n")
				}
				return nil
			}

			report(req.Input, res.OutputPath, res.Notes)
			return nil
		},
	}

	cmd.Flags().StringVar(&size, "size", "", "target the final file size must not exceed, e.g. 80mb")
	cmd.Flags().StringVar(&bitrate, "bitrate", "", "encode at this bitrate, e.g. 128k")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to this exact path instead of the default name")

	return cmd
}
