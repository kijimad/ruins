package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMsgidCoverage は query.T に文字列リテラルで渡す msgid が ja.po に定義済みかを検証する。
// 定義が無いと実行時に黙って英語へフォールバックするため、静的に検知する。
// query は別名 import されないので query.T の構文一致で拾える。
// 非リテラル引数は静的に追えないのでスキップし、件数だけ報告する。
func TestMsgidCoverage(t *testing.T) {
	t.Parallel()

	defined := parsePoMsgids(t)
	root := moduleRoot(t)

	used := map[string]bool{}
	dynamic := 0
	fset := token.NewFileSet()

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Logf("パース失敗をスキップする: %s: %v", path, perr)
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "T" {
				return true
			}
			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "query" {
				return true
			}
			if len(call.Args) < 2 {
				return true
			}
			lit, ok := call.Args[1].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				dynamic++
				return true
			}
			msgid, uerr := strconv.Unquote(lit.Value)
			if uerr != nil || msgid == "" {
				return true
			}
			used[msgid] = true
			return true
		})
		return nil
	})
	require.NoError(t, err)

	var missing []string
	for id := range used {
		if !defined[id] {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)

	t.Logf("静的 msgid %d 種, ja.po 定義 %d 種, 非リテラル引数 %d 箇所は静的検査不可", len(used), len(defined), dynamic)
	require.Empty(t, missing, "ja.po に欠落している msgid:\n%s", strings.Join(missing, "\n"))
}

// moduleRoot は go.mod を上位へ辿ってリポジトリルートを返す。
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod が見つからない")
		dir = parent
	}
}

// parsePoMsgids は埋め込み ja.po から定義済み msgid の集合を作る。
// msgid は複数行に分かれることがあるので、続く "..." 行を連結する。
func parsePoMsgids(t *testing.T) map[string]bool {
	t.Helper()
	data, err := localeFS.ReadFile("locale/ja.po")
	require.NoError(t, err)

	ids := map[string]bool{}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "msgid ") {
			continue
		}
		var b strings.Builder
		b.WriteString(unquotePo(strings.TrimPrefix(line, "msgid ")))
		for i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), `"`) {
			b.WriteString(unquotePo(lines[i+1]))
			i++
		}
		if id := b.String(); id != "" {
			ids[id] = true
		}
	}
	return ids
}

// unquotePo は PO の "..." セグメントを1つ解釈する。PO のエスケープは Go と同じなので Unquote で扱える。
func unquotePo(s string) string {
	s = strings.TrimSpace(s)
	if u, err := strconv.Unquote(s); err == nil {
		return u
	}
	return strings.Trim(s, `"`)
}
