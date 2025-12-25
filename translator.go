package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Subtitle struct {
	Index     int
	Timestamp string
	Text      string
}

const batchSize = 20

func translateSRT(content, from, to string) (string, string, error) {
	subtitles := parseSRT(content)
	if len(subtitles) == 0 {
		return "", "", fmt.Errorf("no subtitles found in file")
	}

	translated := make([]Subtitle, len(subtitles))
	var detectedLang string

	for i := 0; i < len(subtitles); i += batchSize {
		end := min(i+batchSize, len(subtitles))

		batch := subtitles[i:end]

		texts := make([]string, len(batch))
		for j, sub := range batch {
			texts[j] = sub.Text
		}
		combinedText := strings.Join(texts, "\n###SUBTITLE_SEPARATOR###\n")

		translatedText, detected, err := translateText(combinedText, from, to)
		if err != nil {
			return "", "", fmt.Errorf("translation failed at batch starting at %d: %v", i+1, err)
		}

		if i == 0 && detected != "" {
			detectedLang = detected
		}

		translatedTexts := strings.Split(translatedText, "\n###SUBTITLE_SEPARATOR###\n")

		if len(translatedTexts) != len(batch) {
			for j, sub := range batch {
				text, _, err := translateText(sub.Text, from, to)
				if err != nil {
					return "", "", fmt.Errorf("translation failed at subtitle %d: %v", i+j+1, err)
				}
				translated[i+j] = Subtitle{
					Index:     sub.Index,
					Timestamp: sub.Timestamp,
					Text:      text,
				}
			}
		} else {
			for j, sub := range batch {
				translated[i+j] = Subtitle{
					Index:     sub.Index,
					Timestamp: sub.Timestamp,
					Text:      strings.TrimSpace(translatedTexts[j]),
				}
			}
		}

		fmt.Printf("\rTranslating: %d/%d", end, len(subtitles))
	}
	fmt.Println()

	return formatSRT(translated), detectedLang, nil
}

func translateText(text, from, to string) (string, string, error) {
	baseURL := "https://translate.googleapis.com/translate_a/single"
	params := url.Values{}
	params.Add("client", "gtx")
	params.Add("sl", from)
	params.Add("tl", to)
	params.Add("dt", "t")
	params.Add("q", text)

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var result []any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	if len(result) == 0 {
		return "", "", fmt.Errorf("empty translation result")
	}

	translations, ok := result[0].([]any)
	if !ok {
		return "", "", fmt.Errorf("unexpected translation format")
	}

	var translated strings.Builder
	for _, t := range translations {
		if t == nil {
			continue
		}
		parts, ok := t.([]any)
		if !ok || len(parts) == 0 {
			continue
		}
		if parts[0] != nil {
			if str, ok := parts[0].(string); ok {
				translated.WriteString(str)
			}
		}
	}

	var detectedLang string
	if len(result) > 2 && result[2] != nil {
		if lang, ok := result[2].(string); ok {
			detectedLang = lang
		}
	}

	return translated.String(), detectedLang, nil
}

func parseSRT(content string) []Subtitle {
	blocks := regexp.MustCompile(`\n\n+`).Split(strings.TrimSpace(content), -1)
	subtitles := make([]Subtitle, 0)

	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 {
			continue
		}

		index, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil {
			continue
		}

		timestamp := strings.TrimSpace(lines[1])
		text := strings.Join(lines[2:], "\n")

		subtitles = append(subtitles, Subtitle{
			Index:     index,
			Timestamp: timestamp,
			Text:      text,
		})
	}

	return subtitles
}

func formatSRT(subtitles []Subtitle) string {
	var result strings.Builder

	for i, sub := range subtitles {
		if i > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(fmt.Sprintf("%d\n%s\n%s",
			sub.Index, sub.Timestamp, sub.Text))
	}

	return result.String()
}
