package llmutil

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
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
