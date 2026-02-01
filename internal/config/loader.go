package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadedCuratorDocument represents a CuratorDocument loaded from a specific file path.
type LoadedCuratorDocument struct {
	Path     string
	Document *CuratorDocument
}

// LoadCuratorDocuments loads either:
// - a single Curator Document file (YAML), or
// - all Curator Document files in a directory (non-recursive), where files must end in .yaml or .yml.
//
// Directory loads are sorted by filename for stable behavior.
func LoadCuratorDocuments(path string) ([]LoadedCuratorDocument, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		doc, err := loadCuratorDocumentFile(path)
		if err != nil {
			return nil, err
		}
		return []LoadedCuratorDocument{{Path: path, Document: doc}}, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	type candidate struct {
		name string
		path string
	}
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		candidates = append(candidates, candidate{name: name, path: filepath.Join(path, name)})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].name < candidates[j].name
	})

	out := make([]LoadedCuratorDocument, 0, len(candidates))
	for _, c := range candidates {
		doc, err := loadCuratorDocumentFile(c.path)
		if err != nil {
			return nil, err
		}
		out = append(out, LoadedCuratorDocument{Path: c.path, Document: doc})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no curator documents found in directory: %s", path)
	}

	return out, nil
}

func loadCuratorDocumentFile(path string) (*CuratorDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc CuratorDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse curator document %s: %w", path, err)
	}
	return &doc, nil
}
