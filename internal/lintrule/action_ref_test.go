package lintrule_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/lintrule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Dependabot は版の形をした ref と SHA しか更新しない。@beta や @main のような ref は
// 黙って取り残され、古い版のまま動き続ける。workflow の uses を版か SHA に限る
var pinnedActionRef = regexp.MustCompile(`^(v\d+(\.\d+){0,2}|[0-9a-f]{40})$`)

// unpinnedActionRefs は YAML の中の uses のうち、版でも SHA でもない ref を持つものを列挙する。
// リポジトリ内のアクション ./ と docker:// は ref を持たないので対象外にする
func unpinnedActionRefs(src []byte) ([]string, error) {
	var root any
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, err
	}
	var bad []string
	var walk func(v any)
	walk = func(v any) {
		switch n := v.(type) {
		case map[string]any:
			for k, child := range n {
				if s, ok := child.(string); ok && k == "uses" {
					if strings.HasPrefix(s, "./") || strings.HasPrefix(s, "docker://") {
						continue
					}
					_, ref, found := strings.Cut(s, "@")
					if !found || !pinnedActionRef.MatchString(ref) {
						bad = append(bad, s)
					}
					continue
				}
				walk(child)
			}
		case []any:
			for _, child := range n {
				walk(child)
			}
		}
	}
	walk(root)
	return bad, nil
}

func TestUnpinnedActionRefs(t *testing.T) {
	t.Parallel()

	src := []byte(`
jobs:
  a:
    uses: org/repo/.github/workflows/w.yml@main
  b:
    steps:
      - uses: actions/checkout@v7
      - uses: owner/action@v4.9.0
      - uses: owner/action@0123456789abcdef0123456789abcdef01234567
      - uses: anthropics/claude-code-action@beta
      - uses: owner/no-ref
      - uses: ./.github/actions/local
      - uses: docker://alpine:3
`)
	got, err := unpinnedActionRefs(src)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"org/repo/.github/workflows/w.yml@main",
		"anthropics/claude-code-action@beta",
		"owner/no-ref",
	}, got)
}

func TestWorkflows_actionのrefは版かSHA(t *testing.T) {
	t.Parallel()

	root, err := lintrule.ModuleRoot()
	require.NoError(t, err)
	var files []string
	for _, pattern := range []string{".github/workflows/*.yml", ".github/workflows/*.yaml", ".github/actions/*/action.yml"} {
		m, err := filepath.Glob(filepath.Join(root, pattern))
		require.NoError(t, err)
		files = append(files, m...)
	}
	require.NotEmpty(t, files, "workflow が見つからないと検査が素通りする")

	for _, f := range files {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		bad, err := unpinnedActionRefs(src)
		require.NoError(t, err, f)
		assert.Empty(t, bad, f)
	}
}
