// Package rawcheck は raw のデータを rawschema の CUE のスキーマで検証する。
//
// # 責務
//
// 形、値の制約、行の間の参照を、CUE のスキーマに照らして確かめる。違反はデータの位置ごとに Error へ集める。
//
// # 使い分け
//
// CUE のライブラリを持ち込むので、ゲームの本体からは import しない。テストだけが使う。raw は埋め込みの
// データなので、CI のテストで検証すれば足りる。
//
// # 仕様
//
// 検証するのは raws を JSON にした形である。Go の型に読み込んだあとの値を見るので、TOML で省いた
// 0 の値も埋まった状態で検証される。
package rawcheck

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	cuejson "cuelang.org/go/encoding/json"

	"github.com/kijimaD/ruins/internal/rawschema"
)

// Violation は1つの違反。Path はデータの位置で、items.3.melee.accuracy のように . で区切る。
type Violation struct {
	Path    string
	Message string
}

// Error は検証で見つかった違反の集まり。
type Error struct {
	Violations []Violation
}

func (e *Error) Error() string {
	lines := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		lines[i] = v.Path + ": " + v.Message
	}
	return strings.Join(lines, "\n")
}

// Paths は違反のあった位置を、重複を除いて並べて返す。1つの位置に複数の違反があることがある。
func (e *Error) Paths() []string {
	out := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		out[i] = v.Path
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// errSchemaBuild は CUE のスキーマを組めないときに返す
var errSchemaBuild = errors.New("failed to build raw schema")

// checker はコンパイルしたスキーマを持つ。CUE の値は並行に評価できないので mu で直列にする。
type checker struct {
	mu     sync.Mutex
	ctx    *cue.Context
	schema cue.Value
}

var loadChecker = sync.OnceValues(newChecker)

func newChecker() (*checker, error) {
	const dir = "/rawschema"
	overlay := map[string]load.Source{}
	err := fs.WalkDir(rawschema.SchemaFiles, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(rawschema.SchemaFiles, p)
		if err != nil {
			return err
		}
		overlay[path.Join(dir, p)] = load.FromBytes(b)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSchemaBuild, err)
	}
	insts := load.Instances([]string{"."}, &load.Config{Dir: dir, Overlay: overlay})
	if len(insts) != 1 {
		return nil, fmt.Errorf("%w: %d instances", errSchemaBuild, len(insts))
	}
	if err := insts[0].Err; err != nil {
		return nil, fmt.Errorf("%w: %w", errSchemaBuild, err)
	}
	ctx := cuecontext.New()
	schema := ctx.BuildInstance(insts[0])
	if err := schema.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSchemaBuild, err)
	}
	return &checker{ctx: ctx, schema: schema}, nil
}

// Validate は raws を CUE のスキーマで検証する。違反があれば *Error を返す。
func Validate(raws rawschema.Raws) error {
	c, err := loadChecker()
	if err != nil {
		return err
	}
	b, err := json.Marshal(raws)
	if err != nil {
		return err
	}
	expr, err := cuejson.Extract("raws.json", b)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.schema.Unify(c.ctx.BuildExpr(expr))
	if err := v.Validate(cue.Concrete(true)); err != nil {
		return toError(err)
	}
	return nil
}

func toError(err error) error {
	var out Error
	seen := map[Violation]bool{}
	for _, e := range cueerrors.Errors(err) {
		format, args := e.Msg()
		v := Violation{Path: strings.Join(e.Path(), "."), Message: fmt.Sprintf(format, args...)}
		if seen[v] {
			continue
		}
		seen[v] = true
		out.Violations = append(out.Violations, v)
	}
	slices.SortFunc(out.Violations, func(a, b Violation) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Message, b.Message))
	})
	return &out
}
