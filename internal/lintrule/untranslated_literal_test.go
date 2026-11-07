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

	"github.com/stretchr/testify/assert"
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

// untranslatedExemptFiles は開発者向けで訳さない画面のファイル。
var untranslatedExemptFiles = []string{"internal/states/debug_menu.go"}

// TestNoUntranslatedLiteral は画面へ描く文字列に英字の直書きを渡すことを禁止する。
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
		for _, u := range untranslatedLiterals(f) {
			pos := fset.Position(u.pos)
			rel, relErr := filepath.Rel(root, pos.Filename)
			if relErr != nil {
				rel = pos.Filename
			}
			found = append(found, rel+":"+strconv.Itoa(pos.Line)+" "+u.lit)
		}
		return nil
	})
	require.NoError(t, err)

	sort.Strings(found)
	require.Empty(t, found, "画面へ描く文字列は query.T で訳す:\n%s", strings.Join(found, "\n"))
}

// untranslatedUse は画面へ描く文字列に渡された英字の直書き。pos は渡した箇所。
type untranslatedUse struct {
	pos token.Pos
	lit string
}

// untranslatedLiterals は f の中で画面へ描く文字列に渡された英字の直書きを返す。
func untranslatedLiterals(f *ast.File) []untranslatedUse {
	var found []untranslatedUse
	report := func(e ast.Expr) {
		if lit, ok := untranslatedLiteral(e); ok {
			found = append(found, untranslatedUse{pos: e.Pos(), lit: lit})
		}
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
	return found
}

func TestUntranslatedLiterals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"DrawText への直書き", `cv.DrawText(p, "Overworld Map", f, c)`, []string{`"Overworld Map"`}},
		{"直書きを書式にした Sprintf", `cv.DrawText(p, fmt.Sprintf("Chunk %d", n), f, c)`, []string{`"Chunk %d"`}},
		{"直書きで初期化した変数", `header := "Overworld Map"
			cv.DrawText(p, header, f, c)`, []string{`"Overworld Map"`}},
		{"NewText への直書き", `uicore.NewText("Close", f, c)`, []string{`"Close"`}},
		{"OutlinedText への直書き", `hud.OutlinedText(cv, "Hello", f, p, c, c)`, []string{`"Hello"`}},
		{"Choice の Label への直書き", `_ = Choice{Label: "Back"}`, []string{`"Back"`}},
		{"uicore.Text の Value への直書き", `_ = uicore.Text{Value: "Back"}`, []string{`"Back"`}},
		{"訳を通した文字列", `cv.DrawText(p, query.T(world, "Close"), f, c)`, nil},
		{"1文字の語", `cv.DrawText(p, "x F", f, c)`, nil},
		{"略語も訳す", `cv.DrawText(p, "HP", f, c)`, []string{`"HP"`}},
		{"書式の動詞に続く単位", `cv.DrawText(p, fmt.Sprintf("%3dF", n), f, c)`, nil},
		{"画面へ描かない直書き", `log.Print("Overworld Map")`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := parser.ParseFile(token.NewFileSet(), "p.go", "package p\nfunc f() {\n"+tt.body+"\n}\n", 0)
			require.NoError(t, err)
			var got []string
			for _, u := range untranslatedLiterals(f) {
				got = append(got, u.lit)
			}
			assert.Equal(t, tt.want, got)
		})
	}
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

// untranslatedLiteral は e を英字の直書きまで辿れればその直書きを返す。
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

var formatVerb = regexp.MustCompile(`%[-+# 0-9.\[\]*]*[a-zA-Z%]`)

// hasWord は s に2文字以上の英字の語があるかを返す。1文字の語はキーや記号や単位として許す。
func hasWord(s string) bool {
	words := strings.FieldsFunc(formatVerb.ReplaceAllString(s, " "), func(r rune) bool {
		return (r < 'a' || 'z' < r) && (r < 'A' || 'Z' < r)
	})
	return slices.ContainsFunc(words, func(word string) bool {
		return len(word) > 1
	})
}
