
# v2a

A small command-line tool for macOS that extracts audio from MOV videos, compresses MP3 files, and shows media information.

## Installation

On Apple Silicon running macOS 26 or later:

```bash
brew install MaxSiominDev/homebrew-v2a/v2a
```

Homebrew also installs FFmpeg. v2a uses `ffmpeg` and `ffprobe` as external tools; neither is bundled with it.

## Usage

### Extract audio

```bash
v2a extract movie.mov
v2a extract movie.mov --bitrate 192k
v2a extract movie.mov -o ~/Music/movie.mp3
```

The input must be a `.mov` file with exactly one audio stream. By default, v2a uses a high-quality variable bitrate (VBR) preset. Use `--bitrate` for constant bitrate (CBR) encoding.

Metadata and cover art are preserved where MP3 supports them. Audio with more than two channels or a sample rate above 48 kHz is converted to stereo at 48 kHz.

### Compress an MP3

```bash
v2a compress song.mp3 --size 80mb
v2a compress song.mp3 --bitrate 128k
v2a compress song.mp3 --size 80mb -o smaller.mp3
```

Choose either `--size` or `--bitrate`. Compression re-encodes the audio and loses quality, even at the same bitrate. The original file is never modified.

`--size` accepts decimal `kb`, `mb`, and `gb` units, including fractional values, in any letter case. The limit applies to the entire output file, including metadata and cover art.

v2a checks the encoded file's size and retries at lower bitrates until it fits. If it still exceeds the target at 32 kbps, the command fails. Files that already fit the target are left as they are.

### Inspect a file

```bash
v2a info movie.mov
v2a info song.mp3
```

Shows file, video, audio, and metadata details for any file `ffprobe` can read.

### Output files and bitrates

Both `extract` and `compress` save their output next to the input by default.

| Command | Default output |
| --- | --- |
| `v2a extract movie.mov` | `movie.mp3` |
| `v2a compress song.mp3 --size 80mb` | `song.compressed.mp3` |

If the name is taken, v2a adds `(1)`, `(2)`, and so on. An explicit `-o` path is used exactly as given; the command fails if that file already exists.

MP3 supports fixed bitrate steps. Unsupported `--bitrate` values are rounded down to the nearest supported step, with a note printed in the terminal. Values below 32 kbps are rejected.

### Other options

Print the version:

```bash
v2a version
```

Use `-v` to see diagnostics from `ffmpeg` and `ffprobe`:

```bash
v2a -v extract movie.mov
```

## Development

Build from source:

```bash
go build -o v2a ./cmd/v2a
```

Install `ffmpeg` and `ffprobe` separately to run v2a.

Run the checks:

```bash
go test ./...
go test -race ./...
go test -tags integration ./... # requires ffmpeg and ffprobe
golangci-lint run
```

Pushing a `v*` tag runs GoReleaser, publishes an arm64 binary to GitHub Releases, and updates the [Homebrew formula](https://github.com/MaxSiominDev/homebrew-v2a).

## License

[MIT](LICENSE). FFmpeg is not distributed with v2a; its license terms depend on the build you install.