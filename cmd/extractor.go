package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetSubtitleCodec(videoFile string, trackIndex int) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams",
		fmt.Sprintf("s:%d", trackIndex), "-show_entries",
		"stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", videoFile)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	codec := strings.TrimSpace(out.String())
	return codec, nil
}

func ExtractSubtitle(videoFile string, trackIndex int) (string, error) {
	codec, err := GetSubtitleCodec(videoFile, trackIndex)
	if err != nil {
		return "", fmt.Errorf("failed to get subtitle codec: %v", err)
	}

	dir := filepath.Dir(videoFile)
	base := filepath.Base(videoFile)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	outputSRT := filepath.Join(dir, nameWithoutExt)

	var outputFile string
	var cmd *exec.Cmd

	switch codec {
	case "subrip", "ass", "webvtt":
		// Text subtitles: extract as .srt
		outputFile = outputSRT + ".srt"
		cmd = exec.Command("ffmpeg", "-i", videoFile, "-map",
			fmt.Sprintf("0:s:%d", trackIndex), "-c:s", "srt", outputFile)

	case "hdmv_pgs_subtitle":
		// PGS subtitles: extract as .sup (image-based)
		outputFile = outputSRT + ".sup"
		cmd = exec.Command("ffmpeg", "-i", videoFile, "-map",
			fmt.Sprintf("0:s:%d", trackIndex), "-c:s", "copy", outputFile)

	default:
		return "", fmt.Errorf("unsupported subtitle codec: %s", codec)
	}

	fmt.Printf("Extracting track %d with codec %s to %s\n", trackIndex, codec, outputFile)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg extraction failed: %v", err)
	}

	return outputFile, nil
}
