package llmutil

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"text/template"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
	"github.com/gabriel-vasile/mimetype"
)

type CaptionTemplates struct {
	System *template.Template
	Prompt *template.Template
}

type captionPromptData struct {
	Post  *core.PostBlock
	Image *core.ImageBlock
}

func CollectImageBlocks(block *core.PostBlock, includeCommentImages bool, maxImages int) []*core.ImageBlock {
	if block == nil {
		return nil
	}
	images := make([]*core.ImageBlock, 0, len(block.ImageBlocks))
	for i := range block.ImageBlocks {
		images = append(images, &block.ImageBlocks[i])
	}
	if includeCommentImages {
		for ci := range block.Comments {
			for ii := range block.Comments[ci].Images {
				images = append(images, &block.Comments[ci].Images[ii])
			}
		}
	}
	if maxImages > 0 && len(images) > maxImages {
		return images[:maxImages]
	}
	return images
}

func BuildUserMessageWithImages(userPrompt string, images []*core.ImageBlock) llm.Message {
	parts := make([]llm.MessagePart, 0, len(images)+1)
	if userPrompt != "" {
		parts = append(parts, llm.MessagePart{Type: llm.MessagePartText, Text: userPrompt})
	}
	for _, image := range images {
		if image == nil {
			continue
		}
		if url, ok := imageURLForMessage(image); ok {
			parts = append(parts, llm.MessagePart{Type: llm.MessagePartImageURL, ImageURL: url})
		}
	}
	if len(parts) == 0 {
		return llm.Message{Role: llm.RoleUser, Content: userPrompt}
	}
	return llm.Message{Role: llm.RoleUser, Parts: parts}
}

func EnsureImageCaptions(
	ctx context.Context,
	client llm.Client,
	block *core.PostBlock,
	imagesConfig *config.LLMImages,
	templates CaptionTemplates,
	defaultModel string,
	defaultTemp *float64,
	logger *slog.Logger,
) error {
	if imagesConfig == nil || !imagesConfig.Enabled || imagesConfig.Mode != config.ImageModeCaption {
		return nil
	}
	if imagesConfig.Caption == nil {
		return fmt.Errorf("image caption config is required when images.mode=caption")
	}
	if templates.System == nil || templates.Prompt == nil {
		return fmt.Errorf("image caption templates are required when images.mode=caption")
	}

	images := CollectImageBlocks(block, imagesConfig.IncludeCommentImages, imagesConfig.MaxImages)
	if len(images) == 0 {
		return nil
	}

	model := ModelOrDefault(imagesConfig.Caption.Model, defaultModel)
	temperature := imagesConfig.Caption.Temperature
	if temperature == nil {
		temperature = defaultTemp
	}

	captionOne := func(ctx context.Context, image *core.ImageBlock) error {
		if image == nil || image.WasSummarised {
			return nil
		}
		data := captionPromptData{
			Post:  block,
			Image: image,
		}
		systemPrompt, err := ExecuteTemplate(templates.System, data)
		if err != nil {
			return err
		}
		userPrompt, err := ExecuteTemplate(templates.Prompt, data)
		if err != nil {
			return err
		}

		if logger != nil {
			logger.Info("llm image captioning", "image_url", image.URL, "model", model)
		}
		resp, err := ChatSystemUserWithRetries(ctx, client, model, systemPrompt, userPrompt, 3, nil, temperature)
		if err != nil {
			return err
		}
		image.Summary = resp.Content
		image.WasSummarised = true
		return nil
	}

	maxConcurrency := imagesConfig.Caption.MaxConcurrency
	if maxConcurrency <= 1 || len(images) <= 1 {
		for _, image := range images {
			if err := captionOne(ctx, image); err != nil {
				return err
			}
		}
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

captionLoop:
	for _, image := range images {
		image := image
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break captionLoop
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := captionOne(ctx, image); err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
		}()
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func imageURLForMessage(image *core.ImageBlock) (string, bool) {
	if image == nil {
		return "", false
	}
	if image.URL != "" {
		return image.URL, true
	}
	if len(image.ImageData) == 0 {
		return "", false
	}

	contentType := mimetype.Detect(image.ImageData).String()
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(image.ImageData)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return "", false
	}
	encoded := base64.StdEncoding.EncodeToString(image.ImageData)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), true
}

// missingImageMarker is a best-effort signature used to detect a specific class of upstream
// multimodal failures: the LLM provider attempted to fetch an image URL we supplied and got a 404.
//
// Why string matching?
// Some OpenAI-compatible providers (e.g. OpenRouter) surface this as an HTTP 400 response with a
// nested JSON body that contains a human-readable message. In our current llm client abstraction we
// don't get a typed/structured error for "image fetch failed"; we only get an `error`.
//
// When this happens we want to *salvage the post* by retrying without the failing image rather than
// dropping the entire block.
const missingImageMarker = "Received 404 status code when fetching image from URL:"

// MissingImageURL inspects an error and returns (url, true) if it looks like the provider failed to
// fetch an image URL we provided (typically a 404).
//
// This is intentionally conservative:
//   - We only activate on a known marker substring.
//   - If we can extract a valid http(s) URL, we return it so callers can remove that exact image.
//   - If we detect the marker but can't extract a usable URL, we return ("", true) so callers can
//     still treat this as a "missing image" signal and drop *some* image to make progress.
//
// Callers should treat this as a heuristic, not a guarantee.
func MissingImageURL(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	message := err.Error()
	idx := strings.Index(message, missingImageMarker)
	if idx == -1 {
		return "", false
	}
	urlPart := strings.TrimSpace(message[idx+len(missingImageMarker):])
	if urlPart == "" {
		return "", true
	}
	if quote := strings.Index(urlPart, "\""); quote >= 0 {
		urlPart = urlPart[:quote]
	}
	if space := strings.IndexAny(urlPart, " \n\t"); space >= 0 {
		urlPart = urlPart[:space]
	}
	candidate := strings.TrimSpace(urlPart)
	if candidate == "" {
		return "", true
	}
	// Defensive: ensure we only return a usable URL. If parsing fails, callers can still
	// treat this as a missing-image signal and drop an image without relying on a URL match.
	candidate = strings.TrimRight(candidate, ",.)];}")
	parsed, parseErr := url.Parse(candidate)
	if parseErr != nil || parsed == nil {
		return "", true
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", true
	}
	if parsed.Host == "" {
		return "", true
	}
	return candidate, true
}

// DropImageByURL removes a single image from an image slice and returns (remaining, removed).
//
// Behaviour:
//   - If `url` matches an image's URL exactly, that image is removed.
//   - If `url` is empty or doesn't match anything, nothing is removed.
//
// We intentionally do *not* "guess" which image to drop when the URL can't be matched.
// Missing-image detection is heuristic (string parsing). If we can't identify the exact image that
// triggered the provider error, we prefer to fail the block and let block_error_policy decide
// whether to drop the post, rather than silently producing unpredictable results.
func DropImageByURL(images []*core.ImageBlock, url string) ([]*core.ImageBlock, *core.ImageBlock) {
	if len(images) == 0 {
		return images, nil
	}
	if url != "" {
		for i, image := range images {
			if image != nil && image.URL == url {
				remaining := append([]*core.ImageBlock{}, images[:i]...)
				remaining = append(remaining, images[i+1:]...)
				return remaining, image
			}
		}
	}
	return images, nil
}
