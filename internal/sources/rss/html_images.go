package rss

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/bakkerme/curator-ai/internal/core"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ExtractDataURIImagesFromHTML finds <img> tags with data: URIs, extracts the decoded bytes
// into ImageBlocks, and replaces any data: src/srcset attributes with stable placeholder URLs.
//
// placeholderBase should be a small, stable URI prefix (e.g. "curator-image://post/<id>").
// The resulting placeholder is "<placeholderBase>/<index>" where index is 0-based.
func ExtractDataURIImagesFromHTML(htmlText string, placeholderBase string) (string, []core.ImageBlock, error) {
	if htmlText == "" {
		return "", nil, nil
	}
	// Fast path: avoid parsing if clearly no embedded data images.
	lower := strings.ToLower(htmlText)
	if !strings.Contains(lower, "<img") || !strings.Contains(lower, "data:") {
		return htmlText, nil, nil
	}

	// Parse as fragment so this works on partial HTML from RSS feeds.
	ctx := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(strings.NewReader(htmlText), ctx)
	if err != nil {
		return htmlText, nil, fmt.Errorf("failed to parse html fragment: %w", err)
	}

	for _, n := range nodes {
		ctx.AppendChild(n)
	}

	images := make([]core.ImageBlock, 0, 4)
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "img") {
			src := attrValue(n, "src")
			if src == "" {
				// Some feeds use lazy-loading attrs.
				src = firstNonEmpty(attrValue(n, "data-src"), attrValue(n, "data-original"), attrValue(n, "data-lazy-src"))
			}
			if isLikelyDataImage(src) {
				_, data, ok, derr := decodeDataURI(src)
				if derr == nil && ok && len(data) > 0 {
					idx := len(images)
					images = append(images, core.ImageBlock{ImageData: data, WasFetched: true})
					placeholder := fmt.Sprintf("%s/%d", strings.TrimRight(placeholderBase, "/"), idx)

					// Replace the data URI with a small placeholder.
					setAttr(n, "src", placeholder)
					removeAttr(n, "data-src")
					removeAttr(n, "data-original")
					removeAttr(n, "data-lazy-src")

					// Scrub srcset if it contains embedded data.
					if ss := attrValue(n, "srcset"); strings.Contains(strings.ToLower(ss), "data:") {
						removeAttr(n, "srcset")
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(ctx)

	// Render children back into a string (without the artificial <div> wrapper).
	var buf bytes.Buffer
	for c := ctx.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&buf, c)
	}
	out := strings.TrimSpace(buf.String())
	return out, images, nil
}

func isLikelyDataImage(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToLower(s), "data:") {
		return false
	}
	// Most common form: data:image/<type>;base64,...
	return strings.Contains(strings.ToLower(s), "data:image/")
}

func decodeDataURI(s string) (mediaType string, data []byte, ok bool, err error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToLower(s), "data:") {
		return "", nil, false, nil
	}

	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return "", nil, false, nil
	}
	meta := s[len("data:"):comma]
	payload := s[comma+1:]

	metaLower := strings.ToLower(meta)
	isBase64 := strings.Contains(metaLower, ";base64")
	if semi := strings.IndexByte(meta, ';'); semi >= 0 {
		mediaType = meta[:semi]
	} else {
		mediaType = meta
	}
	mediaType = strings.TrimSpace(mediaType)

	if isBase64 {
		payload = strings.TrimSpace(payload)
		payload = strings.ReplaceAll(payload, "\n", "")
		payload = strings.ReplaceAll(payload, "\r", "")

		var (
			decoded     []byte
			derrStd     error
			derrRawStd  error
			derrRawURL  error
		)

		// Try standard base64 encoding first.
		decoded, derrStd = base64.StdEncoding.DecodeString(payload)
		if derrStd == nil {
			return mediaType, decoded, true, nil
		}

		// Some producers omit padding.
		decoded, derrRawStd = base64.RawStdEncoding.DecodeString(payload)
		if derrRawStd == nil {
			return mediaType, decoded, true, nil
		}

		// Some producers use URL-safe base64.
		decoded, derrRawURL = base64.RawURLEncoding.DecodeString(payload)
		if derrRawURL == nil {
			return mediaType, decoded, true, nil
		}

		// All decoding strategies failed: return an error that summarizes all attempts.
		return mediaType, nil, false, fmt.Errorf("base64 decode failed; StdEncoding: %v; RawStdEncoding: %v; RawURLEncoding: %v", derrStd, derrRawStd, derrRawURL)
	}

	unescaped, uerr := url.PathUnescape(payload)
	if uerr != nil {
		return mediaType, nil, false, uerr
	}
	return mediaType, []byte(unescaped), true, nil
}

func attrValue(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	if n == nil {
		return
	}
	for i := range n.Attr {
		if strings.EqualFold(n.Attr[i].Key, key) {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func removeAttr(n *html.Node, key string) {
	if n == nil {
		return
	}
	out := n.Attr[:0]
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			continue
		}
		out = append(out, a)
	}
	n.Attr = out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
