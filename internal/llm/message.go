package llm

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/nostalgia296/ocs-ai/internal/image"
)

// Base64Image represents a base64-encoded image with metadata.
type Base64Image struct {
	Label string
	URL   string
	Data  string
}

// Message represents a chat message.
type Message struct {
	Role    string
	Content interface{}
}

// BuildMultimodalMessages builds multimodal message list and returns base64 images.
// apiProtocol controls the content block format: "anthropic" uses Anthropic-style image blocks,
// all others use Chat Completions (OpenAI-compatible) style.
func BuildMultimodalMessages(ctx context.Context, prompt string, apiProtocol string, imageURLs []string, imageItems []map[string]string, includeLabels bool, httpClient *http.Client) ([]Message, []Base64Image, bool) {
	imageURLs = imageURLs[:]
	imageItems = imageItems[:]
	useImages := len(imageURLs) > 0

	var base64Images []Base64Image
	if useImages {
		imageSources := imageItems
		if len(imageSources) == 0 {
			imageSources = make([]map[string]string, len(imageURLs))
			for i, url := range imageURLs {
				imageSources[i] = map[string]string{"url": url, "label": fmt.Sprintf("Image %d", i+1)}
			}
		}

		for _, item := range imageSources {
			imgURL := item["url"]
			if imgURL == "" {
				continue
			}
			b64, err := image.DownloadAsBase64(ctx, httpClient, imgURL)
			if err != nil {
				continue
			}
			base64Images = append(base64Images, Base64Image{
				Label:  item["label"],
				URL:    imgURL,
				Data:   b64,
			})
		}

		if len(base64Images) == 0 {
			useImages = false
		}
	}

	if !useImages {
		return []Message{{Role: "user", Content: prompt}}, nil, false
	}

	return buildMultimodalWithImages(prompt, "", base64Images, includeLabels, apiProtocol)
}

func buildMultimodalWithImages(prompt, systemContent string, base64Images []Base64Image, includeLabels bool, apiProtocol string) ([]Message, []Base64Image, bool) {
	isAnthropic := apiProtocol == "anthropic"
	userContent := []interface{}{}

	// Helper to build an image content block appropriate for the protocol
	buildImageBlock := func(img Base64Image) interface{} {
		if isAnthropic {
			mediaType, data := parseDataURI(img.Data)
			return map[string]interface{}{
				"type": "image",
				"source": map[string]string{
					"type":       "base64",
					"media_type": mediaType,
					"data":       data,
				},
			}
		}
		return map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]string{"url": img.Data},
		}
	}

	if len(base64Images) > 0 {
		canInterleave := true
		urlPattern := buildURLPattern(base64Images)
		remainingOccurrences := make([]Base64Image, len(base64Images))
		copy(remainingOccurrences, base64Images)
		cursor := 0

		for _, match := range regexp.MustCompile(urlPattern).FindAllStringIndex(prompt, -1) {
			textSegment := prompt[cursor:match[0]]
			if strings.TrimSpace(textSegment) != "" {
				userContent = append(userContent, map[string]string{"type": "text", "text": strings.TrimSpace(textSegment)})
			}

			matchedURL := prompt[match[0]:match[1]]
			imageIndex := -1
			for idx, bi := range remainingOccurrences {
				if bi.URL == matchedURL {
					imageIndex = idx
					break
				}
			}
			if imageIndex == -1 {
				canInterleave = false
				break
			}

			imageItem := remainingOccurrences[imageIndex]
			remainingOccurrences = append(remainingOccurrences[:imageIndex], remainingOccurrences[imageIndex+1:]...)

			if includeLabels {
				userContent = append(userContent, map[string]string{"type": "text", "text": "[" + imageItem.Label + "]"})
			}
			userContent = append(userContent, buildImageBlock(imageItem))
			cursor = match[1]
		}

		if canInterleave {
			textSegment := prompt[cursor:]
			if strings.TrimSpace(textSegment) != "" {
				userContent = append(userContent, map[string]string{"type": "text", "text": strings.TrimSpace(textSegment)})
			}
			return []Message{
				{Role: "system", Content: systemContent},
				{Role: "user", Content: userContent},
			}, base64Images, true
		}
	}

	// Traditional mode: images first, then text
	userContent = []interface{}{}
	for _, img := range base64Images {
		if includeLabels {
			userContent = append(userContent, map[string]string{"type": "text", "text": "[" + img.Label + "]"})
		}
		userContent = append(userContent, buildImageBlock(img))
	}
	userContent = append(userContent, map[string]string{"type": "text", "text": prompt})

	return []Message{
		{Role: "system", Content: systemContent},
		{Role: "user", Content: userContent},
	}, base64Images, true
}

func buildURLPattern(base64Images []Base64Image) string {
	if len(base64Images) == 0 {
		return ""
	}
	urls := make([]string, len(base64Images))
	for i, img := range base64Images {
		urls[i] = regexp.QuoteMeta(img.URL)
	}
	return strings.Join(urls, "|")
}

// parseDataURI extracts media type and raw base64 data from a data URI.
// Input: "data:image/jpeg;base64,<base64data>"
// Returns: "image/jpeg", "<base64data>"
func parseDataURI(dataURI string) (mediaType, data string) {
	// Strip "data:" prefix
	if !strings.HasPrefix(dataURI, "data:") {
		return "image/jpeg", dataURI
	}
	rest := dataURI[5:]
	// Find the comma separating header from data
	commaIdx := strings.Index(rest, ",")
	if commaIdx == -1 {
		return "image/jpeg", rest
	}
	header := rest[:commaIdx]
	data = rest[commaIdx+1:]
	// Extract media type: "image/jpeg;base64" -> "image/jpeg"
	semiIdx := strings.Index(header, ";")
	if semiIdx != -1 {
		mediaType = header[:semiIdx]
	} else {
		mediaType = header
	}
	if mediaType == "" {
		mediaType = "image/jpeg"
	}
	return
}
