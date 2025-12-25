package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func ExtractSubtitles(videoFile string, trackIndex int) (string, error) {
	dir := filepath.Dir(videoFile)
	base := filepath.Base(videoFile)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))

	outputSRT := filepath.Join(dir, nameWithoutExt+".srt")

	cmd := exec.Command("ffmpeg", "-i", videoFile, "-map",
		fmt.Sprintf("0:s:%d", trackIndex), outputSRT)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg extraction failed: %v", err)
	}

	return outputSRT, nil
}
