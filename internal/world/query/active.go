package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ActiveFilterN は現ステージのエンティティだけを対象にするフィルタを返す。
//
// 共存方式では全ステージのエンティティが同一 world にいる。ステージ跨ぎのシステムが
// 退避中ステージのエンティティを巻き込むと、描画・視界・AI・座標索引が混ざる。
// この関数群は Suspended の除外を1箇所に集約し、各システムが付け忘れないようにする。
// 追加の除外は exclude で渡す。生の ecs.NewFilterN でなくこれを使う。

// ActiveFilter1 は現ステージ限定の Filter1 を返す。
func ActiveFilter1[A any](world w.World, exclude ...ecs.Comp) *ecs.Filter1[A] {
	return ecs.NewFilter1[A](world.ECS).Without(withSuspended(exclude)...)
}

// ActiveFilter2 は現ステージ限定の Filter2 を返す。
func ActiveFilter2[A, B any](world w.World, exclude ...ecs.Comp) *ecs.Filter2[A, B] {
	return ecs.NewFilter2[A, B](world.ECS).Without(withSuspended(exclude)...)
}

// ActiveFilter3 は現ステージ限定の Filter3 を返す。
func ActiveFilter3[A, B, C any](world w.World, exclude ...ecs.Comp) *ecs.Filter3[A, B, C] {
	return ecs.NewFilter3[A, B, C](world.ECS).Without(withSuspended(exclude)...)
}

// ActiveFilter4 は現ステージ限定の Filter4 を返す。
func ActiveFilter4[A, B, C, D any](world w.World, exclude ...ecs.Comp) *ecs.Filter4[A, B, C, D] {
	return ecs.NewFilter4[A, B, C, D](world.ECS).Without(withSuspended(exclude)...)
}

// withSuspended は追加除外の先頭に Suspended を足す。除外集合の単一の情報源
func withSuspended(exclude []ecs.Comp) []ecs.Comp {
	return append([]ecs.Comp{ecs.C[gc.Suspended]()}, exclude...)
}
