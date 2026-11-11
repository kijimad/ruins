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

	"github.com/kijimaD/ruins/internal/lintrule"

	"github.com/stretchr/testify/require"
)

// TestMsgidCoverage は query.T に文字列リテラルで渡す msgid と、キー束縛の Label が ja.po に定義済みかを検証する。
// 定義が無いと実行時に黙って英語へフォールバックするため、静的に検知する。
// query は別名 import されないので query.T の構文一致で拾える。
// 非リテラル引数は静的に追えないのでスキップし、件数だけ報告する。
func TestMsgidCoverage(t *testing.T) {
	t.Parallel()

	defined := parsePoMsgids(t)
	root, err := lintrule.ModuleRoot()
	require.NoError(t, err)

	used := map[string]bool{}
	dynamic := 0
	fset := token.NewFileSet()

	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		// ビルドが通れば必ずパースできる。失敗は検査漏れになるのでハードエラーにする
		require.NoError(t, perr, "パースできない: %s", path)
		ast.Inspect(f, func(n ast.Node) bool {
			for _, label := range bindingLabels(n) {
				used[label] = true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// 第2引数が msgid になる呼び出しを拾う。query.T はその場で訳し、
			// Cancel は reason を CancelReason に積んで activity_manager が query.T で訳す。
			if !isMsgidCall(call.Fun) || len(call.Args) < 2 {
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

// isMsgidCall は呼び出しの第2引数が msgid になる関数かを返す。
// query.T はその場で訳す。activity の中断の関数は reason を CancelReason に積み、表示するときに
// query.T で訳すため、その reason も msgid として ja.po 存在を検証する。
func isMsgidCall(fun ast.Expr) bool {
	switch fn := fun.(type) {
	case *ast.SelectorExpr:
		if pkg, ok := fn.X.(*ast.Ident); ok && pkg.Name == "query" && fn.Sel.Name == "T" {
			return true
		}
		return cancelFuncs[fn.Sel.Name]
	case *ast.Ident:
		return cancelFuncs[fn.Name]
	}
	return false
}

// cancelFuncs は第2引数に中断の理由の msgid を取る activity の関数
var cancelFuncs = map[string]bool{
	"Cancel":            true,
	"CancelActivity":    true,
	"Interrupt":         true,
	"InterruptActivity": true,
}

// bindingLabels は n が keybind.Binding の表か値なら、直書きの Label を返す。Label は keybind.HintEntries が query.T で訳す。
func bindingLabels(n ast.Node) []string {
	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var elems []ast.Expr
	switch typ := lit.Type.(type) {
	case *ast.ArrayType:
		if !isBindingType(typ.Elt) {
			return nil
		}
		elems = lit.Elts
	default:
		if !isBindingType(typ) {
			return nil
		}
		elems = []ast.Expr{lit}
	}
	var labels []string
	for _, e := range elems {
		b, ok := e.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, elt := range b.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Label" {
				continue
			}
			if v, ok := kv.Value.(*ast.BasicLit); ok && v.Kind == token.STRING {
				if s, err := strconv.Unquote(v.Value); err == nil && s != "" {
					labels = append(labels, s)
				}
			}
		}
	}
	return labels
}

func isBindingType(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.SelectorExpr:
		return e.Sel.Name == "Binding"
	case *ast.Ident:
		return e.Name == "Binding"
	}
	return false
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
