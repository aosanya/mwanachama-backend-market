package market_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func markdownPaths(t *testing.T) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && p != "." {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil || len(paths) == 0 {
		t.Fatalf("want markdown files, got %v %v", paths, err)
	}
	return paths
}

func TestTheDocsCallNoPosterASupplier(t *testing.T) {
	for _, p := range markdownPaths(t) {
		for _, tok := range strings.FieldsFunc(strings.ToLower(readFile(t, p)), func(r rune) bool { return r < 'a' || r > 'z' }) {
			if tok == "supplier" || tok == "suppliers" {
				t.Errorf("%s carries the word %q; a poster is a shop, a buyer or a rider", p, tok)
				break
			}
		}
	}
}
