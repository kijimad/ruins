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

// TestGoGenerateOnlyInGenGo は //go:generate をルートの gen.go にだけ書かせ、raw の型の生成を先頭に置かせる。
// go generate ./... はパッケージのパスの順に走るので、ほかのパッケージに書くと実行順が gen.go の並びから外れる。
// 後の生成はアプリをビルドして動くので、raw の型を先に書き出さないと古い型でビルドされる。
func TestGoGenerateOnlyInGenGo(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)

	directives := map[string][]string{}
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
		rel, relErr := filepath.Rel(root, path)
		require.NoError(t, relErr)
		for line := range strings.Lines(string(src)) {
			// cmd/go は //go:generate の後に空白かタブが続く行をディレクティブとみなす
			if rest, ok := strings.CutPrefix(line, "//go:generate"); ok && (strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "\t")) {
				directives[rel] = append(directives[rel], strings.TrimSpace(rest))
			}
		}
		return nil
	})
	require.NoError(t, err)

	got := make([]string, 0, len(directives))
	for f := range directives {
		got = append(got, f)
	}
	sort.Strings(got)
	require.Equal(t, []string{"gen.go"}, got, "//go:generate はルートの gen.go に実行順で書く")
	assert.Equal(t, "go run ./internal/rawschema/gen internal/rawschema", directives["gen.go"][0], "raw の型の生成を先頭に置く")
}
