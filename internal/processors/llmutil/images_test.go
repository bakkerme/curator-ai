package llmutil

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
)

type mockLLMClient struct {
	chatCompletion func(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error)
}

func (m *mockLLMClient) ChatCompletion(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	if m.chatCompletion == nil {
		return llm.ChatResponse{Content: "ok"}, nil
	}
	return m.chatCompletion(ctx, request)
}

func TestCollectImageBlocks_NilBlock(t *testing.T) {
	imgs := CollectImageBlocks(nil, true, 10)
	if imgs != nil {
		t.Fatalf("expected nil, got %v", imgs)
	}
}

func TestCollectImageBlocks_CollectsPostAndCommentImagesAndRespectsMax(t *testing.T) {
	block := &core.PostBlock{
		ImageBlocks: []core.ImageBlock{{URL: "post-1"}, {URL: "post-2"}},
		Comments: []core.CommentBlock{
			{Images: []core.ImageBlock{{URL: "c1-1"}, {URL: "c1-2"}}},
			{Images: []core.ImageBlock{{URL: "c2-1"}}},
		},
	}

	t.Run("exclude comment images", func(t *testing.T) {
		imgs := CollectImageBlocks(block, false, 0)
		if len(imgs) != 2 {
			t.Fatalf("expected 2 images, got %d", len(imgs))
		}
		if imgs[0].URL != "post-1" || imgs[1].URL != "post-2" {
			t.Fatalf("unexpected urls: %q, %q", imgs[0].URL, imgs[1].URL)
		}
	})

	t.Run("include comment images", func(t *testing.T) {
		imgs := CollectImageBlocks(block, true, 0)
		if len(imgs) != 5 {
			t.Fatalf("expected 5 images, got %d", len(imgs))
		}
		got := []string{imgs[0].URL, imgs[1].URL, imgs[2].URL, imgs[3].URL, imgs[4].URL}
		want := []string{"post-1", "post-2", "c1-1", "c1-2", "c2-1"}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("unexpected order at %d: got %q want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("max images truncates", func(t *testing.T) {
		imgs := CollectImageBlocks(block, true, 3)
		if len(imgs) != 3 {
			t.Fatalf("expected 3 images, got %d", len(imgs))
		}
		got := []string{imgs[0].URL, imgs[1].URL, imgs[2].URL}
		want := []string{"post-1", "post-2", "c1-1"}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("unexpected order at %d: got %q want %q", i, got[i], want[i])
			}
		}
	})
}

func TestBuildUserMessageWithImages(t *testing.T) {
	t.Run("nil and empty inputs", func(t *testing.T) {
		msg := BuildUserMessageWithImages("", nil)
		if msg.Role != llm.RoleUser {
			t.Fatalf("expected role user, got %q", msg.Role)
		}
		if msg.Content != "" {
			t.Fatalf("expected empty content, got %q", msg.Content)
		}
		if len(msg.Parts) != 0 {
			t.Fatalf("expected no parts, got %d", len(msg.Parts))
		}
	})

	t.Run("text then image parts, skips nil images", func(t *testing.T) {
		images := []*core.ImageBlock{nil, {URL: "https://example.com/a.png"}, {URL: "https://example.com/b.png"}}
		msg := BuildUserMessageWithImages("hello", images)
		if msg.Role != llm.RoleUser {
			t.Fatalf("expected role user, got %q", msg.Role)
		}
		if msg.Content != "" {
			t.Fatalf("expected empty content when parts present, got %q", msg.Content)
		}
		if len(msg.Parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(msg.Parts))
		}
		if msg.Parts[0].Type != llm.MessagePartText || msg.Parts[0].Text != "hello" {
			t.Fatalf("unexpected first part: %#v", msg.Parts[0])
		}
		if msg.Parts[1].Type != llm.MessagePartImageURL || msg.Parts[1].ImageURL != "https://example.com/a.png" {
			t.Fatalf("unexpected second part: %#v", msg.Parts[1])
		}
		if msg.Parts[2].Type != llm.MessagePartImageURL || msg.Parts[2].ImageURL != "https://example.com/b.png" {
			t.Fatalf("unexpected third part: %#v", msg.Parts[2])
		}
	})
}

func TestEnsureImageCaptions_GuardsAndBasicBehavior(t *testing.T) {
	ctx := context.Background()
	client := &mockLLMClient{}
	block := &core.PostBlock{}

	t.Run("disabled does nothing", func(t *testing.T) {
		cfg := &config.LLMImages{Enabled: false, Mode: config.ImageModeCaption, Caption: &config.LLMImageCaption{}}
		err := EnsureImageCaptions(ctx, client, block, cfg, CaptionTemplates{}, "m", nil, nil)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("missing caption config errors", func(t *testing.T) {
		cfg := &config.LLMImages{Enabled: true, Mode: config.ImageModeCaption}
		err := EnsureImageCaptions(ctx, client, block, cfg, CaptionTemplates{System: template.Must(template.New("s").Parse("s")), Prompt: template.Must(template.New("p").Parse("p"))}, "m", nil, nil)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing templates errors", func(t *testing.T) {
		cfg := &config.LLMImages{Enabled: true, Mode: config.ImageModeCaption, Caption: &config.LLMImageCaption{}}
		err := EnsureImageCaptions(ctx, client, block, cfg, CaptionTemplates{}, "m", nil, nil)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("no images no-ops", func(t *testing.T) {
		cfg := &config.LLMImages{Enabled: true, Mode: config.ImageModeCaption, Caption: &config.LLMImageCaption{MaxConcurrency: 2}}
		tmpls := CaptionTemplates{
			System: template.Must(template.New("s").Parse("system")),
			Prompt: template.Must(template.New("p").Parse("prompt")),
		}
		err := EnsureImageCaptions(ctx, client, &core.PostBlock{}, cfg, tmpls, "m", nil, nil)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

func TestEnsureImageCaptions_CaptionsAndRespectsIncludeAndMaxImages(t *testing.T) {
	ctx := context.Background()

	var calls atomic.Int32
	client := &mockLLMClient{chatCompletion: func(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
		calls.Add(1)
		return llm.ChatResponse{Content: "caption:" + request.Messages[1].Content}, nil
	}}

	block := &core.PostBlock{
		ImageBlocks: []core.ImageBlock{{URL: "post-1"}},
		Comments:    []core.CommentBlock{{Images: []core.ImageBlock{{URL: "c1-1"}, {URL: "c1-2"}}}},
	}

	cfg := &config.LLMImages{
		Enabled:              true,
		Mode:                 config.ImageModeCaption,
		IncludeCommentImages: true,
		MaxImages:            2,
		Caption:              &config.LLMImageCaption{MaxConcurrency: 1},
	}

	tmpls := CaptionTemplates{
		System: template.Must(template.New("s").Parse("sys")),
		Prompt: template.Must(template.New("p").Parse("{{.Image.URL}}")),
	}

	err := EnsureImageCaptions(ctx, client, block, cfg, tmpls, "m", nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if calls.Load() != 2 {
		t.Fatalf("expected 2 caption calls, got %d", calls.Load())
	}
	if !block.ImageBlocks[0].WasSummarised || block.ImageBlocks[0].Summary == "" {
		t.Fatalf("expected post image to be summarised")
	}
	if !block.Comments[0].Images[0].WasSummarised || block.Comments[0].Images[0].Summary == "" {
		t.Fatalf("expected first comment image to be summarised")
	}
	if block.Comments[0].Images[1].WasSummarised {
		t.Fatalf("expected second comment image to be skipped due to max_images")
	}
}

func TestEnsureImageCaptions_ConcurrentCaptioningRespectsMaxConcurrency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	startedCh := make(chan struct{}, 10)
	unblock := make(chan struct{})

	client := &mockLLMClient{chatCompletion: func(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
		cur := inFlight.Add(1)
		for {
			prev := maxInFlight.Load()
			if cur <= prev {
				break
			}
			if maxInFlight.CompareAndSwap(prev, cur) {
				break
			}
		}
		select {
		case startedCh <- struct{}{}:
		default:
		}
		select {
		case <-unblock:
		case <-ctx.Done():
			inFlight.Add(-1)
			return llm.ChatResponse{}, ctx.Err()
		}
		inFlight.Add(-1)
		return llm.ChatResponse{Content: "ok"}, nil
	}}

	block := &core.PostBlock{
		ImageBlocks: []core.ImageBlock{{URL: "a"}, {URL: "b"}, {URL: "c"}},
	}
	maxConc := 2
	cfg := &config.LLMImages{
		Enabled: true,
		Mode:    config.ImageModeCaption,
		Caption: &config.LLMImageCaption{MaxConcurrency: maxConc},
	}
	tmpls := CaptionTemplates{
		System: template.Must(template.New("s").Parse("sys")),
		Prompt: template.Must(template.New("p").Parse("{{.Image.URL}}")),
	}

	// Release once we observe at least 2 concurrent starts.
	go func() {
		seen := 0
		for seen < maxConc {
			select {
			case <-startedCh:
				seen++
			case <-ctx.Done():
				return
			}
		}
		close(unblock)
	}()

	err := EnsureImageCaptions(ctx, client, block, cfg, tmpls, "m", nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if maxInFlight.Load() < 2 {
		t.Fatalf("expected some concurrency (maxInFlight>=2), got %d", maxInFlight.Load())
	}
	if maxInFlight.Load() > int32(maxConc) {
		t.Fatalf("expected max concurrency <= %d, got %d", maxConc, maxInFlight.Load())
	}
	for i := range block.ImageBlocks {
		if !block.ImageBlocks[i].WasSummarised {
			t.Fatalf("expected image %d to be summarised", i)
		}
	}
}

func TestImageURLForMessage_DataURL_UsesDetectedMimeType(t *testing.T) {
	t.Run("png", func(t *testing.T) {
		img := &core.ImageBlock{ImageData: []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, // some extra bytes
		}}
		url, ok := imageURLForMessage(img)
		if !ok {
			t.Fatalf("expected ok=true")
		}
		if !strings.HasPrefix(url, "data:image/png;base64,") {
			t.Fatalf("expected png data url prefix, got %q", url)
		}
	})

	t.Run("jpeg", func(t *testing.T) {
		img := &core.ImageBlock{ImageData: []byte{
			0xFF, 0xD8, 0xFF, 0xE0, // JPEG SOI + JFIF marker-ish
			0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, // "JFIF\x00"
		}}
		url, ok := imageURLForMessage(img)
		if !ok {
			t.Fatalf("expected ok=true")
		}
		if !strings.HasPrefix(url, "data:image/jpeg;base64,") {
			t.Fatalf("expected jpeg data url prefix, got %q", url)
		}
	})
}

func TestImageURLForMessage_ReturnsFalseForUnknownBytes(t *testing.T) {
	img := &core.ImageBlock{ImageData: []byte("not-an-image")}
	_, ok := imageURLForMessage(img)
	if ok {
		t.Fatalf("expected ok=false")
	}
}

func TestImageURLForMessage_UsesURLWhenPresent(t *testing.T) {
	img := &core.ImageBlock{URL: "https://example.com/a.jpg", ImageData: []byte("not-an-image")}
	url, ok := imageURLForMessage(img)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if url != img.URL {
		t.Fatalf("expected %q, got %q", img.URL, url)
	}
}
