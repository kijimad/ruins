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
// 生の ecs.NewFilterN でなくこれを使う。付け忘れは silent な leak になる。
//
// 追加の除外は返り値へ Without を重ねればよい。ark の Without は除外を累積するので、
// Suspended に上乗せされる。例: ActiveFilter2[A, B](world).Without(ecs.C[gc.Dead]())。
//
// arity 1..4 の重複は ark 自身が Filter1..4 に分かれているのと同じ Go の制約による。
// 可変長の型パラメータが無いので1関数に畳めない。

// suspendedExclude は Suspended だけを除外する共有スライス。
// spread で渡すと毎フレームのホットパスでも再アロケーションを起こさない。
// Without は読み取り専用で受け取るため共有して安全
var suspendedExclude = []ecs.Comp{ecs.C[gc.Suspended]()}

// ActiveFilter1 は現ステージ限定の Filter1 を返す。
func ActiveFilter1[A any](world w.World) *ecs.Filter1[A] {
	return ecs.NewFilter1[A](world.ECS).Without(suspendedExclude...)
}

// ActiveFilter2 は現ステージ限定の Filter2 を返す。
func ActiveFilter2[A, B any](world w.World) *ecs.Filter2[A, B] {
	return ecs.NewFilter2[A, B](world.ECS).Without(suspendedExclude...)
}

// ActiveFilter3 は現ステージ限定の Filter3 を返す。
func ActiveFilter3[A, B, C any](world w.World) *ecs.Filter3[A, B, C] {
	return ecs.NewFilter3[A, B, C](world.ECS).Without(suspendedExclude...)
}

// ActiveFilter4 は現ステージ限定の Filter4 を返す。
func ActiveFilter4[A, B, C, D any](world w.World) *ecs.Filter4[A, B, C, D] {
	return ecs.NewFilter4[A, B, C, D](world.ECS).Without(suspendedExclude...)
}
