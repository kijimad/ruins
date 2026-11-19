package lintrule_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoGenerateOnlyInGenGo は //go:generate をルートの gen.go にだけ書かせる。go generate ./... は
// パッケージのパスの順に走るので、ほかのパッケージに書くと実行順が gen.go の並びから外れる。
func TestGoGenerateOnlyInGenGo(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)

	files := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, rerr := os.ReadFile(path)
		require.NoError(t, rerr)
		for line := range strings.Lines(string(src)) {
			if strings.HasPrefix(line, "//go:generate ") {
				rel, relErr := filepath.Rel(root, path)
				require.NoError(t, relErr)
				files[rel] = true
				break
			}
		}
		return nil
	})
	require.NoError(t, err)

	got := make([]string, 0, len(files))
	for f := range files {
		got = append(got, f)
	}
	sort.Strings(got)
	assert.Equal(t, []string{"gen.go"}, got, "//go:generate はルートの gen.go に実行順で書く")
}
