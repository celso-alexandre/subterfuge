# Subterfuge

A fast and simple CLI tool for translating SRT subtitle files between languages using Google Translate.

## Features

- 🌍 Translate subtitles to any language with auto-detection
- 📝 Preserves original timing
- 🚀 Batch translation for improved speed
- 💯 Free - uses Google Translate's public API
- ⚡ Progress indicator during translation
- 🔄 Two modes: create new file or replace original

## Installation

```bash
go install github.com/celso-alexandre/subterfuge@latest
```

Or build from source:

```bash
git clone https://github.com/celso-alexandre/subterfuge.git
cd subterfuge
go build
```

## Usage

```bash
subterfuge [command] [options] <input.srt>
```

### Commands

- `extract` - Extract SRT file out of video file (requires ffmpeg)
- `translate` - Translate the given SRT file

### Options

- `-s <lang>` - Source language (default: `auto` for auto-detection)
- `-t <lang>` - Target language (default: `en`)
- `-m <mode>` - Output mode: `create` or `replace` (default: `create`)

### Examples

Extract SRT subtitle from video:

```bash
subterfuge extract movie.mp4
# Creates: movie.srt
```

Translate to Portuguese (replace mode):

```bash
subterfuge translate -t pt -m replace movie.srt
# Renames original to: movie.<input-lang>.srt
```

Translate to Spanish, creating a new file:

```bash
subterfuge translate -t es -m create movie.srt
# Creates: movie.es.srt
```

Translate from French to Japanese:

```bash
subterfuge translate -s fr -t ja -m create anime.srt
# Creates: anime.ja.srt
```

## Translate Modes

### Create Mode (default)

- Creates new file: `filename.<output-lang>.srt`
- Compatible with VLC and other media players' auto-detection

### Replace Mode

- Renames original file to `filename.<input-lang>.srt`
- Replaces input file with translated content

## Language Codes

Use ISO 639-1 language codes:

| Language            | Code    | Language | Code |
| ------------------- | ------- | -------- | ---- |
| English             | `en`    | Spanish  | `es` |
| Portuguese          | `pt`    | French   | `fr` |
| Portuguese (Brazil) | `pt-br` | German   | `de` |
| Italian             | `it`    | Japanese | `ja` |
| Korean              | `ko`    | Chinese  | `zh` |
| Russian             | `ru`    | Arabic   | `ar` |

[Full list of language codes](https://cloud.google.com/translate/docs/languages)

## How It Works

0. (Optional) Extracts SRT from video using ffmpeg
1. Parses the input SRT file into subtitle blocks
2. Auto-detects source language if not specified
3. Outputs translated SRT file with original timing preserved
4. Shows real-time progress during translation

## Limitations

- Uses unofficial Google Translate API (rate limits may apply)
- Includes small delays between batches to avoid rate limiting
- Best for personal/non-commercial use

## Contributing

Contributions welcome! Please open an issue or submit a pull request.

## License

MIT
