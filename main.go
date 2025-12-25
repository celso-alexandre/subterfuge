package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	from := flag.String("s", "auto", "Source language")
	to := flag.String("t", "en", "Target language")
	mode := flag.String("m", "create", "Output mode: 'create' or 'replace'")
	// force := flag.Bool("force", false, "Force translation even if source equals target")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Error: input file required")
		fmt.Println("Usage: subterfuge [options] <input.srt>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	input := args[0]

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
	translated, detectedLang, err := translateSRT(string(content), *from, *to)
	if err != nil {
		fmt.Printf("Error translating: %v\n", err)
		os.Exit(1)
	}

	sourceLang := *from
	if sourceLang == "auto" && detectedLang != "" {
		sourceLang = detectedLang
	}

	// if !*force && sourceLang != "auto" && normalizeLanguageCode(sourceLang) == normalizeLanguageCode(*to) {
	// 	fmt.Printf("Error: source language (%s) is the same as target language (%s)\n", sourceLang, *to)
	// 	fmt.Println("Use -force flag to translate anyway")
	// 	os.Exit(1)
	// }

	output, err := handleOutputMode(input, sourceLang, *to, *mode, translated)
	if err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Translated file saved to %s\n", output)
}

func normalizeLanguageCode(code string) string {
	parts := strings.Split(strings.ToLower(code), "-")
	return parts[0]
}

func buildSubtitlePath(dir, nameWithoutExt, ext, lang string) string {
	normalizedLang := normalizeLanguageCode(lang)
	return filepath.Join(dir, nameWithoutExt+"."+normalizedLang+ext)
}

func extractFilenameParts(path string) (dir, nameWithoutExt, ext string) {
	dir = filepath.Dir(path)
	base := filepath.Base(path)
	ext = filepath.Ext(base)
	nameWithoutExt = strings.TrimSuffix(base, ext)
	return
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func handleOutputMode(input, sourceLang, targetLang, mode, content string) (string, error) {
	dir, nameWithoutExt, ext := extractFilenameParts(input)

	targetPath := buildSubtitlePath(dir, nameWithoutExt, ext, targetLang)
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
		currentNormalizedName := buildSubtitlePath(dir, nameWithoutExt, ext, sourceLang)

		if input == currentNormalizedName {
			if err := moveFile(tempPath, input); err != nil {
				return "", fmt.Errorf("failed to write translated file: %v", err)
			}
			return input, nil
		}

		sourcePath := buildSubtitlePath(dir, nameWithoutExt, ext, sourceLang)
		if _, err := os.Stat(sourcePath); err == nil {
			return "", fmt.Errorf("cannot rename: %s already exists", sourcePath)
		}

		if err := moveFile(tempPath, targetPath); err != nil {
			return "", fmt.Errorf("failed to write translated file: %v", err)
		}

		return targetPath, nil
	}

	if err := moveFile(tempPath, targetPath); err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}

	return targetPath, nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		if copyErr := copyFile(src, dst); copyErr != nil {
			return copyErr
		}
		os.Remove(src)
	}
	return nil
}
