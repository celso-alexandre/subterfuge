package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/celso-alexandre/subterfuge/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "translate":
		translateCommand(os.Args[2:])
	case "extract":
		extractCommand(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: subterfuge <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  translate    Translate an SRT file between languages")
	fmt.Println("  extract      Extract subtitles from a video file")
	fmt.Println("\nUse 'subterfuge <command> -h' for more information about a command")
}

func translateCommand(args []string) {
	fs := flag.NewFlagSet("translate", flag.ExitOnError)
	from := fs.String("f", "auto", "Source language")
	to := fs.String("t", "en", "Target language")
	mode := fs.String("m", "create", "Output mode: 'replace' or 'create'")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Println("Error: input file required")
		fmt.Println("Usage: subterfuge translate [options] <input.srt>")
		fs.PrintDefaults()
		os.Exit(1)
	}

	input := fs.Arg(0)

	if *mode != "replace" && *mode != "create" {
		fmt.Println("Error: mode must be 'replace' or 'create'")
		os.Exit(1)
	}

	fmt.Printf("Reading %s...\n", input)
	content, err := os.ReadFile(input)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Translating from %s to %s...\n", *from, *to)
	translated, detectedLang, err := cmd.TranslateSRT(string(content), *from, *to)
	if err != nil {
		fmt.Printf("Error translating: %v\n", err)
		os.Exit(1)
	}

	sourceLang := *from
	if sourceLang == "auto" && detectedLang != "" {
		sourceLang = detectedLang
	}

	output, err := handleOutputMode(input, sourceLang, *to, *mode, translated)
	if err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Translated file saved to %s\n", output)
}

func extractCommand(args []string) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Println("Error: video file required")
		fmt.Println("Usage: subterfuge extract <video_file>")
		os.Exit(1)
	}

	videoFile := fs.Arg(0)

	fmt.Printf("Extracting subtitles from %s...\n", videoFile)
	outputSRT, err := cmd.ExtractSubtitles(videoFile, 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Subtitles extracted to %s\n", outputSRT)
}

func handleOutputMode(input, sourceLang, targetLang, mode, content string) (string, error) {
	dir, nameWithoutExt, ext := cmd.ExtractFilenameParts(input)

	targetPath := cmd.BuildSubtitlePath(dir, nameWithoutExt, ext, targetLang)
	if _, err := os.Stat(targetPath); err == nil {
		return "", fmt.Errorf("destination file already exists: %s", targetPath)
	}

	tempFile, err := os.CreateTemp("", "subterfuge-*.srt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()

	parts := strings.Split(nameWithoutExt, ".")
	if len(parts) > 1 {
		lastPart := parts[len(parts)-1]
		if len(lastPart) >= 2 && len(lastPart) <= 5 {
			nameWithoutExt = strings.Join(parts[:len(parts)-1], ".")
		}
	}

	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to write to temp file: %v", err)
	}
	tempFile.Close()

	defer func() {
		if _, err := os.Stat(tempPath); err == nil {
			os.Remove(tempPath)
		}
	}()

	if mode == "replace" {
		currentNormalizedName := cmd.BuildSubtitlePath(dir, nameWithoutExt, ext, sourceLang)

		if input == currentNormalizedName {
			if err := cmd.MoveFile(tempPath, input); err != nil {
				return "", fmt.Errorf("failed to write translated file: %v", err)
			}
			return input, nil
		}

		sourcePath := cmd.BuildSubtitlePath(dir, nameWithoutExt, ext, sourceLang)
		if _, err := os.Stat(sourcePath); err == nil {
			return "", fmt.Errorf("cannot rename: %s already exists", sourcePath)
		}

		if err := cmd.MoveFile(tempPath, targetPath); err != nil {
			return "", fmt.Errorf("failed to write translated file: %v", err)
		}

		return targetPath, nil
	}

	if err := cmd.MoveFile(tempPath, targetPath); err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}

	return targetPath, nil
}
