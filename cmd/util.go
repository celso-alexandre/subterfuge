package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func NormalizeLanguageCode(code string) string {
	parts := strings.Split(strings.ToLower(code), "-")
	return parts[0]
}

func BuildSubtitlePath(dir, nameWithoutExt, ext, lang string) string {
	normalizedLang := NormalizeLanguageCode(lang)
	return filepath.Join(dir, nameWithoutExt+"."+normalizedLang+ext)
}

func ExtractFilenameParts(path string) (dir, nameWithoutExt, ext string) {
	dir = filepath.Dir(path)
	base := filepath.Base(path)
	ext = filepath.Ext(base)
	nameWithoutExt = strings.TrimSuffix(base, ext)
	return
}

func CopyFile(src, dst string) error {
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

func MoveFile(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		if copyErr := CopyFile(src, dst); copyErr != nil {
			return copyErr
		}
		os.Remove(src)
	}
	return nil
}
