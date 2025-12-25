# Subterfuge

A fast and simple CLI tool for translating SRT subtitle files between languages using Google Translate.

## Features

- 🌍 Translate subtitles to any language with auto-detection
- 📝 Preserves original timing and formatting
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
subterfuge [options] <input.srt>
```

### Options

- `-s <lang>` - Source language (default: `auto` for auto-detection)
- `-t <lang>` - Target language (default: `en`)
- `-m <mode>` - Output mode: `create` or `replace` (default: `create`)

### Examples

Translate to Portuguese (replace mode):

```bash
subterfuge -t pt -m replace movie.srt
```

Translate to Spanish, creating a new file:

```bash
subterfuge -t es -m create movie.srt
# Creates: movie.es.srt
```

Translate from French to Japanese:

```bash
subterfuge -s fr -t ja -m create anime.srt
# Creates: anime.ja.srt
```

## Modes

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
