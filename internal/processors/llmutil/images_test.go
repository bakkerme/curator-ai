package llmutil

import (
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/core"
)

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
