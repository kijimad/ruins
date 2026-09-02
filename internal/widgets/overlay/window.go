package overlay

import (
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// detailRowsPerPage は詳細モーダル1ページに収める性能行の数。
// 行数でページ分割することで、短い項目は1ページに収まり、行の多い項目だけがはみ出さないよう分割される
const detailRowsPerPage = 12

// DetailPageCount はエンティティの詳細モーダルのページ数を返す。
// 呼び出し側が実体からページ数を確かめる公開の入口
func DetailPageCount(world w.World, entity ecs.Entity) int {
	return detailPageCount(len(entityspec.SpecRows(world, entity)))
}

// detailPageCount は行数からページ数を返す。ページ計算は pagination に委ねる。
// 行が無いか負数のときの1丸めも pagination.GetTotalPages が吸収する
func detailPageCount(rowCount int) int {
	return pagination.New(0, rowCount, detailRowsPerPage).GetTotalPages()
}
