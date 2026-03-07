package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Subtitle struct {
	Index     int
	Timestamp string
	Text      string
}

const batchSize = 50
const maxParallelRequests = 2

func StripHTMLTags(text string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(text, "")
}

func TranslateSRT(content, from, to string) (string, string, error) {
	subtitles := ParseSRT(content)
	if len(subtitles) == 0 {
		return "", "", fmt.Errorf("no subtitles found in file")
	}

	translated := make([]Subtitle, len(subtitles))

	type batchJob struct {
		start int
		end   int
	}

	jobs := make(chan batchJob)
	errCh := make(chan error, 1)

	var doneCount int64

	var detectedLang string
	var detectedOnce sync.Once
	setDetected := func(lang string) {
		if lang == "" {
			return
		}
		detectedOnce.Do(func() {
			detectedLang = lang
		})
	}

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()

		sep := "\n###SUBTITLE_SEPARATOR###\n"

		for job := range jobs {
			batch := subtitles[job.start:job.end]

			texts := make([]string, len(batch))
			for j, sub := range batch {
				texts[j] = StripHTMLTags(sub.Text)
			}
			combinedText := strings.Join(texts, sep)

			translatedText, detected, err := TranslateText(
				combinedText,
				from,
				to,
			)
			if err != nil {
				if strings.Contains(err.Error(), "413") {
					for j, sub := range batch {
						text, det, errSingle := TranslateText(
							sub.Text,
							from,
							to,
						)
						if errSingle != nil {
							select {
							case errCh <- fmt.Errorf(
								"translation failed at subtitle %d: %v",
								job.start+j+1,
								errSingle,
							):
							default:
							}
							return
						}

						setDetected(det)

						translated[job.start+j] = Subtitle{
							Index:     sub.Index,
							Timestamp: sub.Timestamp,
							Text:      text,
						}

						n := atomic.AddInt64(&doneCount, 1)
						fmt.Printf("\rTranslating: %d/%d", n, len(subtitles))
					}
					continue
				}

				select {
				case errCh <- fmt.Errorf(
					"translation failed at batch starting at %d: %v",
					job.start+1,
					err,
				):
				default:
				}
				return
			}

			setDetected(detected)

			translatedTexts := strings.Split(translatedText, sep)

			if len(translatedTexts) != len(batch) {
				for j, sub := range batch {
					text, det, errSingle := TranslateText(sub.Text, from, to)
					if errSingle != nil {
						select {
						case errCh <- fmt.Errorf(
							"translation failed at subtitle %d: %v",
							job.start+j+1,
							errSingle,
						):
						default:
						}
						return
					}

					setDetected(det)

					translated[job.start+j] = Subtitle{
						Index:     sub.Index,
						Timestamp: sub.Timestamp,
						Text:      text,
					}

					n := atomic.AddInt64(&doneCount, 1)
					fmt.Printf("\rTranslating: %d/%d", n, len(subtitles))
				}
				continue
			}

			for j, sub := range batch {
				translated[job.start+j] = Subtitle{
					Index:     sub.Index,
					Timestamp: sub.Timestamp,
					Text:      strings.TrimSpace(translatedTexts[j]),
				}

				n := atomic.AddInt64(&doneCount, 1)
				fmt.Printf("\rTranslating: %d/%d", n, len(subtitles))
			}
		}
	}

	wg.Add(maxParallelRequests)
	for i := 0; i < maxParallelRequests; i++ {
		go worker()
	}

	go func() {
		defer close(jobs)
		for i := 0; i < len(subtitles); i += batchSize {
			end := min(i+batchSize, len(subtitles))
			jobs <- batchJob{start: i, end: end}
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		fmt.Println()
		return "", "", err
	case <-done:
		fmt.Println()
		return FormatSRT(translated), detectedLang, nil
	}
}

func TranslateText(text, from, to string) (string, string, error) {
	baseURL := "https://translate.googleapis.com/translate_a/single"
	params := url.Values{}
	params.Add("client", "gtx")
	params.Add("sl", from)
	params.Add("tl", to)
	params.Add("dt", "t")
	params.Add("q", text)

	reqURL := baseURL + "?" + params.Encode()

	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	const maxRetries = 10

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := client.Get(reqURL)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				backoffSleep(attempt)
				continue
			}
			return "", "", err
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if readErr != nil {
			lastErr = readErr
			if attempt < maxRetries {
				backoffSleep(attempt)
				continue
			}
			return "", "", readErr
		}

		if resp.StatusCode == 429 || (resp.StatusCode >= 500 &&
			resp.StatusCode <= 599) {
			lastErr = fmt.Errorf("API returned status %d", resp.StatusCode)
			if attempt < maxRetries {
				backoffSleep(attempt)
				continue
			}
			return "", "", lastErr
		}

		if resp.StatusCode != 200 {
			return "", "", fmt.Errorf("API returned status %d", resp.StatusCode)
		}

		var result []any
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = err
			if attempt < maxRetries {
				backoffSleep(attempt)
				continue
			}
			return "", "", err
		}

		if len(result) == 0 {
			lastErr = fmt.Errorf("empty translation result")
			if attempt < maxRetries {
				backoffSleep(attempt)
				continue
			}
			return "", "", lastErr
		}

		translations, ok := result[0].([]any)
		if !ok {
			lastErr = fmt.Errorf("unexpected translation format")
			if attempt < maxRetries {
				backoffSleep(attempt)
				continue
			}
			return "", "", lastErr
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

	return "", "", lastErr
}

func backoffSleep(attempt int) {
	d := time.Duration(attempt) * 250 * time.Millisecond
	if d > 3*time.Second {
		d = 3 * time.Second
	}
	time.Sleep(d)
}

func ParseSRT(content string) []Subtitle {
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

func FormatSRT(subtitles []Subtitle) string {
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
