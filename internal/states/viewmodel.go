package states

import (
	"sort"

	"github.com/mlange-42/ark/ecs"
)

// sortByNameID はビューモデル列を name と entity.ID() の昇順で並べる。ark の反復順や query の未ソートに
// 依存しない決定的な表示順を与える。同名の並びは entity.ID() で固定する。ID は並び決めにだけ使い、
// 表示やゴールデンには出さない
func sortByNameID[T any](items []T, key func(T) (string, ecs.Entity)) {
	sort.Slice(items, func(i, j int) bool {
		ni, ei := key(items[i])
		nj, ej := key(items[j])
		if ni != nj {
			return ni < nj
		}
		return ei.ID() < ej.ID()
	})
}
