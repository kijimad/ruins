package lintrule_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// textArgs は画面へ文字列を描く呼び出しの名前と、描く文字列の引数の位置。
var textArgs = map[string]int{
	"DrawText":     1,
	"NewText":      0,
	"OutlinedText": 1,
}

// textFields は画面へ描く文字列を持つ構造体と、そのフィールド名。
var textFields = map[string]string{
	"Text":   "Value",
	"Choice": "Label",
}

// neutralWords は訳さずに描いてよい英字の語。1文字の語はキーや記号や単位なので常に許す。
var neutralWords = []string{"HP", "Esc", "Enter"}

// untranslatedExemptFiles は開発者向けで訳さない画面のファイル。
var untranslatedExemptFiles = []string{"internal/states/debug_menu.go"}

// TestNoUntranslatedLiteral は画面へ描く文字列に英字の直書きを渡すことを禁止する。
// 描く文字列は query.T で訳す。直書き、直書きの書式の fmt.Sprintf、それらで初期化した変数を拾う。
// 実際に描かれた文字列の検査は states の TestUntranslatedText が持ち、こちらは描かない画面も覆う。
func TestNoUntranslatedLiteral(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)
	fset := token.NewFileSet()
	var found []string

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if rel, relErr := filepath.Rel(root, path); relErr == nil && slices.Contains(untranslatedExemptFiles, rel) {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, perr, "パースできない: %s", path)
		report := func(e ast.Expr) {
			lit, ok := untranslatedLiteral(e)
			if !ok {
				return
			}
			pos := fset.Position(e.Pos())
			rel, relErr := filepath.Rel(root, pos.Filename)
			if relErr != nil {
				rel = pos.Filename
			}
			found = append(found, rel+":"+strconv.Itoa(pos.Line)+" "+lit)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if i, ok := textArgs[calleeName(n.Fun)]; ok && i < len(n.Args) {
					report(n.Args[i])
				}
			case *ast.CompositeLit:
				field, ok := textFields[calleeName(n.Type)]
				if !ok {
					return true
				}
				for _, elt := range n.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if key, ok := kv.Key.(*ast.Ident); ok && key.Name == field {
						report(kv.Value)
					}
				}
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)

	sort.Strings(found)
	require.Empty(t, found, "画面へ描く文字列は query.T で訳す:\n%s", strings.Join(found, "\n"))
}

// calleeName は f(...) や pkg.f(...) の関数名、T{} や pkg.T{} の型名を返す。
func calleeName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

// untranslatedLiteral は e が訳を通らない英字の文字列ならその直書きを返す。
// 直書き、直書きを書式にした fmt.Sprintf、それらで初期化したローカル変数を辿る。
func untranslatedLiteral(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		if err != nil || !hasWord(s) {
			return "", false
		}
		return e.Value, true
	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Sprintf" || len(e.Args) == 0 {
			return "", false
		}
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "fmt" {
			return "", false
		}
		return untranslatedLiteral(e.Args[0])
	case *ast.Ident:
		if e.Obj == nil {
			return "", false
		}
		switch decl := e.Obj.Decl.(type) {
		case *ast.AssignStmt:
			for i, lhs := range decl.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && id.Name == e.Name && i < len(decl.Rhs) {
					return untranslatedLiteral(decl.Rhs[i])
				}
			}
		case *ast.ValueSpec:
			for i, id := range decl.Names {
				if id.Name == e.Name && i < len(decl.Values) {
					return untranslatedLiteral(decl.Values[i])
				}
			}
		}
	}
	return "", false
}

// formatVerb は書式の動詞。語に数えない。
var formatVerb = regexp.MustCompile(`%[-+# 0-9.\[\]*]*[a-zA-Z%]`)

// hasWord は s に neutralWords 以外の2文字以上の英字の語があるかを返す。
func hasWord(s string) bool {
	words := strings.FieldsFunc(formatVerb.ReplaceAllString(s, " "), func(r rune) bool {
		return (r < 'a' || 'z' < r) && (r < 'A' || 'Z' < r)
	})
	return slices.ContainsFunc(words, func(word string) bool {
		return len(word) > 1 && !slices.Contains(neutralWords, word)
	})
}
