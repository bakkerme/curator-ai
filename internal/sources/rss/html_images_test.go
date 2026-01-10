package rss

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/core"
)

func TestExtractDataURIImagesFromHTML_Empty(t *testing.T) {
	out, images, err := ExtractDataURIImagesFromHTML("", "curator-image://post/abc")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
	if images != nil {
		t.Fatalf("expected nil images, got %#v", images)
	}
}

func TestExtractDataURIImagesFromHTML_FastPath_NoImgOrNoData(t *testing.T) {
	t.Run("no img tag", func(t *testing.T) {
		in := `<p>Hello</p>`
		out, images, err := ExtractDataURIImagesFromHTML(in, "curator-image://post/abc")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if out != in {
			t.Fatalf("expected pass-through %q, got %q", in, out)
		}
		if images != nil {
			t.Fatalf("expected nil images, got %#v", images)
		}
	})

	t.Run("img but no data", func(t *testing.T) {
		in := `<p><img src="https://example.com/a.png"></p>`
		out, images, err := ExtractDataURIImagesFromHTML(in, "curator-image://post/abc")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if out != in {
			t.Fatalf("expected pass-through %q, got %q", in, out)
		}
		if images != nil {
			t.Fatalf("expected nil images, got %#v", images)
		}
	})
}

func TestExtractDataURIImagesFromHTML_SingleValidDataURI_ReplacesAndExtracts(t *testing.T) {
	data := []byte("hello")
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)

	in := `<p>before <img src="` + dataURI + `"> after</p>`
	base := "curator-image://post/abc"

	out, images, err := ExtractDataURIImagesFromHTML(in, base)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	wantURL := base + "/0"
	if images[0].URL != wantURL {
		t.Fatalf("expected image URL %q, got %q", wantURL, images[0].URL)
	}
	if !images[0].WasFetched {
		t.Fatalf("expected WasFetched=true")
	}
	if !bytes.Equal(images[0].ImageData, data) {
		t.Fatalf("expected image data %q, got %q", string(data), string(images[0].ImageData))
	}

	if !strings.Contains(out, `src="`+wantURL+`"`) {
		t.Fatalf("expected output to contain placeholder src %q, got %q", wantURL, out)
	}
	if strings.Contains(out, dataURI) {
		t.Fatalf("expected output to not contain original data URI, got %q", out)
	}
}

func TestExtractDataURIImagesFromHTML_MultipleImages_StablePlaceholders(t *testing.T) {
	data1 := []byte("one")
	data2 := []byte("two")
	uri1 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data1)
	uri2 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data2)

	in := `<div><img src="` + uri1 + `"><span>x</span><img src="` + uri2 + `"></div>`
	base := "curator-image://post/abc/"

	out, images, err := ExtractDataURIImagesFromHTML(in, base)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}

	want0 := "curator-image://post/abc/0"
	want1 := "curator-image://post/abc/1"

	if images[0].URL != want0 || images[1].URL != want1 {
		t.Fatalf("expected placeholder URLs %q and %q, got %q and %q", want0, want1, images[0].URL, images[1].URL)
	}
	if !bytes.Equal(images[0].ImageData, data1) {
		t.Fatalf("image[0] data mismatch")
	}
	if !bytes.Equal(images[1].ImageData, data2) {
		t.Fatalf("image[1] data mismatch")
	}

	if !strings.Contains(out, `src="`+want0+`"`) || !strings.Contains(out, `src="`+want1+`"`) {
		t.Fatalf("expected output to contain both placeholders, got %q", out)
	}
	if strings.Contains(out, uri1) || strings.Contains(out, uri2) {
		t.Fatalf("expected output to not contain original data URIs, got %q", out)
	}
}

func TestExtractDataURIImagesFromHTML_LazyLoadingAttrs_UsesDataSrc_RemovesLazyAttrs(t *testing.T) {
	data := []byte("lazy")
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)

	in := `<p><img data-src="` + dataURI + `" data-original="x" data-lazy-src="y"></p>`
	base := "curator-image://post/abc"

	out, images, err := ExtractDataURIImagesFromHTML(in, base)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	want := base + "/0"
	if !strings.Contains(out, `src="`+want+`"`) {
		t.Fatalf("expected output to contain placeholder src %q, got %q", want, out)
	}
	if strings.Contains(out, "data-src") || strings.Contains(out, "data-original") || strings.Contains(out, "data-lazy-src") {
		t.Fatalf("expected lazy-loading attrs to be removed, got %q", out)
	}
}

func TestExtractDataURIImagesFromHTML_InvalidBase64_DoesNotReplace(t *testing.T) {
	// Invalid base64 should not be replaced; function should not error.
	bad := "data:image/png;base64,NOT-BASE64!!!"
	in := `<p><img src="` + bad + `"></p>`

	out, images, err := ExtractDataURIImagesFromHTML(in, "curator-image://post/abc")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("expected 0 images, got %d", len(images))
	}

	// Rendering may normalize the fragment, so check containment instead of exact equality.
	if !strings.Contains(out, bad) {
		t.Fatalf("expected output to still contain original data URI, got %q", out)
	}
	if strings.Contains(out, `curator-image://post/abc/0`) {
		t.Fatalf("did not expect placeholder replacement, got %q", out)
	}
}

func TestExtractDataURIImagesFromHTML_ScrubsSrcsetWhenContainsData(t *testing.T) {
	data := []byte("srcset")
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)

	in := `<img src="` + dataURI + `" srcset="data:image/png;base64,AAAA 1x, https://example.com/x.png 2x">`
	base := "curator-image://post/abc"

	out, images, err := ExtractDataURIImagesFromHTML(in, base)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	want := base + "/0"
	if !strings.Contains(out, `src="`+want+`"`) {
		t.Fatalf("expected output to contain placeholder src %q, got %q", want, out)
	}
	if strings.Contains(strings.ToLower(out), "srcset=") {
		t.Fatalf("expected srcset to be removed when it contains embedded data, got %q", out)
	}
}

func TestExtractDataURIImagesFromHTML_IgnoresNonImageDataURIs(t *testing.T) {
	// isLikelyDataImage requires "data:image/", so non-image data URIs should be left alone.
	data := []byte("hi")
	dataURI := "data:text/plain;base64," + base64.StdEncoding.EncodeToString(data)

	in := `<p><img src="` + dataURI + `"></p>`
	out, images, err := ExtractDataURIImagesFromHTML(in, "curator-image://post/abc")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("expected 0 images, got %d", len(images))
	}
	// Because the fast-path sees "<img" and "data:", it will parse+render, so assert containment.
	if !strings.Contains(out, dataURI) {
		t.Fatalf("expected output to contain original non-image data URI, got %q", out)
	}
}

func TestExtractDataURIImagesFromHTML_ImageBlockShape(t *testing.T) {
	// Ensure we don't accidentally regress the ImageBlock fields that downstream code expects.
	data := []byte("shape")
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	in := `<img src="` + dataURI + `">`

	out, images, err := ExtractDataURIImagesFromHTML(in, "curator-image://post/abc")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = out

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	var _ core.ImageBlock = images[0] // compile-time check the type is what we expect
	if images[0].URL == "" {
		t.Fatalf("expected image URL to be set")
	}
	if len(images[0].ImageData) == 0 {
		t.Fatalf("expected image data to be set")
	}
	if !images[0].WasFetched {
		t.Fatalf("expected WasFetched=true")
	}
}
