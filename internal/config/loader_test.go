package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadCuratorDocuments_File(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "curator.yaml")
	writeFile(t, path, []byte(`workflow:
  name: "Example"
  trigger: []
  sources: []
  output: []
`))

	loaded, err := LoadCuratorDocuments(path)
	if err != nil {
		t.Fatalf("LoadCuratorDocuments returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 document, got %d", len(loaded))
	}
	if loaded[0].Path != path {
		t.Fatalf("expected path %q, got %q", path, loaded[0].Path)
	}
	if loaded[0].Document == nil {
		t.Fatalf("expected document to be non-nil")
	}
	if loaded[0].Document.Workflow.Name != "Example" {
		t.Fatalf("expected workflow.name to be %q, got %q", "Example", loaded[0].Document.Workflow.Name)
	}
}

func TestLoadCuratorDocuments_Directory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.yml"), []byte(`workflow:
  name: "B"
  trigger: []
  sources: []
  output: []
`))
	writeFile(t, filepath.Join(dir, "a.yaml"), []byte(`workflow:
  name: "A"
  trigger: []
  sources: []
  output: []
`))
	writeFile(t, filepath.Join(dir, "not-a-doc.txt"), []byte("nope"))
	writeFile(t, filepath.Join(dir, ".ignored.yaml"), []byte(`workflow:
  name: "Ignored"
  trigger: []
  sources: []
  output: []
`))

	loaded, err := LoadCuratorDocuments(dir)
	if err != nil {
		t.Fatalf("LoadCuratorDocuments returned error: %v", err)
	}

	paths := make([]string, 0, len(loaded))
	for _, l := range loaded {
		paths = append(paths, filepath.Base(l.Path))
	}

	want := []string{"a.yaml", "b.yml"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("unexpected loaded order: got %v, want %v", paths, want)
	}
}

func TestLoadCuratorDocuments_DirectoryEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := LoadCuratorDocuments(dir)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadCuratorDocuments_InvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	writeFile(t, path, []byte("workflow: ["))

	_, err := LoadCuratorDocuments(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
