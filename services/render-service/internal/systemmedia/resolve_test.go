package systemmedia

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogMatchesContracts(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "packages", "contracts", "mvp-constants.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read parity file: %v", err)
	}
	var expected struct {
		SystemMedia struct {
			StorageKeyPrefix string `json:"storageKeyPrefix"`
			Items            map[string]struct {
				Filename string `json:"filename"`
				Kind     string `json:"kind"`
			} `json:"items"`
		} `json:"systemMedia"`
	}
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cat := Catalog()
	if len(cat) != len(expected.SystemMedia.Items) {
		t.Fatalf("catalog size: got %d want %d", len(cat), len(expected.SystemMedia.Items))
	}
	for id, meta := range expected.SystemMedia.Items {
		got, ok := cat[id]
		if !ok {
			t.Fatalf("missing catalog id %q", id)
		}
		if got.Filename != meta.Filename || got.Kind != meta.Kind {
			t.Fatalf("catalog %q mismatch: %+v vs %+v", id, got, meta)
		}
		wantKey := expected.SystemMedia.StorageKeyPrefix + "/" + id
		if StorageKey(id) != wantKey {
			t.Fatalf("StorageKey(%q) = %q want %q", id, StorageKey(id), wantKey)
		}
	}
}

func TestResolvePackedPlaceholders(t *testing.T) {
	root := filepath.Join("..", "..", "system-media")
	paths, err := ResolveAll(root)
	if err != nil {
		t.Fatalf("ResolveAll: %v", err)
	}
	for id, p := range paths {
		st, err := os.Stat(p)
		if err != nil || st.Size() == 0 {
			t.Fatalf("packed system media %q missing or empty at %s", id, p)
		}
	}
}

func TestResolveUnknownID(t *testing.T) {
	_, err := Resolve(".", "not-a-real-media")
	if err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestResolveMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Resolve(dir, "flash")
	if err == nil {
		t.Fatal("expected missing file error")
	}
}
