package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// deployOffsets は展開に開けた場所を要求する周囲8マス。斜めも含む。ここが塞がれていると展開を拒否し、
// 開けた地形を探す判断を生む。
var deployOffsets = []consts.Coord[consts.Tile]{
	{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
	{X: -1, Y: 0}, {X: 1, Y: 0},
	{X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
}

// defaultCubeCargoItem は展開の効果を示すため最初から畳んで入れておく貨物。
const defaultCubeCargoItem = "garlic_bread"

// defaultCubeCargoOffset は既定貨物を展開で出す相対位置。左上へ置いて往復が目に見えるようにする。
var defaultCubeCargoOffset = consts.Coord[consts.Tile]{X: -1, Y: -1}

// StowDefaultCubeCargo はキューブに既定の貨物を1つ畳み込んで入れる。展開すると左上に現れ、
// 圧縮で畳まれる往復が一目で分かる。ゲーム開始時のキューブ生成でだけ呼び、キューブは初期から圧縮で始まる。
func StowDefaultCubeCargo(world w.World, cube ecs.Entity) error {
	item, err := spawnItemBase(world, defaultCubeCargoItem)
	if err != nil {
		return err
	}
	MoveToStowed(world, item, cube, defaultCubeCargoOffset)
	return nil
}

// DeployCube はキューブを展開状態にする。周囲8マスが開けていれば Deployed を付け、畳み込んでいた貨物を
// 元の相対位置へ出し直して true を返す。1タイルでも壁や敵、物で塞がれていれば状態を変えず false を返す。
func DeployCube(world w.World, cube ecs.Entity) bool {
	if world.Components.Deployed.Has(cube) {
		return true
	}
	if !deploySpaceFree(world, cube) {
		return false
	}
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	releaseStowedItems(world, cube)
	return true
}

// deploySpaceFree は周囲8マスが展開に使えるかを返す。壁・敵・フィールドのアイテムや prop があれば偽。
// メニューは隣接で開くのでプレイヤーは周囲のどれかに立つ。展開を妨げないよう本人は除外する。
func deploySpaceFree(world w.World, cube ecs.Entity) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}
	player, _ := query.GetPlayerEntity(world)
	base := world.Components.GridElement.Get(cube).Coord
	tiles := make(map[consts.Coord[consts.Tile]]bool, len(deployOffsets))
	for _, off := range deployOffsets {
		t := base.Add(off)
		if si.IsBlockPass(t) {
			return false
		}
		if e, ok := si.CharacterAt(t); ok && e != player {
			return false
		}
		tiles[t] = true
	}
	// フィールドのアイテムや prop が周囲8マスにあれば展開しない。空間索引は BlockPass と character しか
	// 持たないので LocationOnField を走査する。見つけたら Close でロックを解いて早期に返す
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if e == cube || !world.Components.GridElement.Has(e) {
			continue
		}
		if tiles[world.Components.GridElement.Get(e).Coord] {
			q.Close()
			return false
		}
	}
	return true
}

// StowCube はキューブを圧縮状態へ戻す。野営の貨物を相対位置ごと畳み込む。
func StowCube(world w.World, cube ecs.Entity) {
	if !world.Components.Deployed.Has(cube) {
		return
	}
	stowNearbyItems(world, cube)
	world.Components.Deployed.Remove(cube)
}

// stowNearbyItems は野営内のフィールドアイテムをキューブへ畳み込む。Stowed を付けて燃料と区別し、
// キューブからの相対位置を覚えて展開で同じ配置へ戻せるようにする。容量に入る分だけ取り込み、
// 入りきらないものはその場に残す。キャラクターやキューブ自身は対象外。
// 畳み込む範囲は野営 CubeDeployCampRadius で、展開判定の周囲8マス、すなわちチェビシェフ距離1より広い。
// 展開は開けた足場を1マス分だけ要求し、圧縮は野営に広げた物を余さず回収するので、範囲が非対称でよい。
func stowNearbyItems(world w.World, cube ecs.Entity) {
	if !world.Components.WeightCapacity.Has(cube) {
		return
	}
	maxCap := world.Components.WeightCapacity.Get(cube).Max
	// キャッシュの WeightCapacity.Current は WeightDirtySystem が非同期に更新するため、同一バッチの
	// 追加を反映しない。実測の収納重量を基準にし、取り込むたびに加算して容量を正しく判定する
	used := query.CubeWeight(world, cube)
	base := world.Components.GridElement.Get(cube).Coord

	// 畳み込むのはアイテムだけ。草・木・岩・遺跡入口などの prop も LocationOnField を持つが、
	// Item は持たない。Item に限らないと野営周りの地物まで吸い込んでしまう
	var items []ecs.Entity
	q := ecs.NewFilter2[gc.Item, gc.LocationOnField](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if e == cube || !world.Components.GridElement.Has(e) {
			continue
		}
		if chebyshev(world.Components.GridElement.Get(e).Coord, base) <= consts.CubeDeployCampRadius {
			items = append(items, e)
		}
	}
	for _, item := range items {
		iw := query.GetEntityWeight(world, item)
		if used+iw > maxCap {
			continue
		}
		offset := world.Components.GridElement.Get(item).Sub(base)
		MoveToStowed(world, item, cube, offset)
		used += iw
	}
}

// releaseStowedItems は畳み込んだ貨物を、記録した相対位置へ出し直す。燃料タンクの中身は残す。
// キューブが移動していても相対位置で戻すので、置いた配置を保ったまま野営を再現する。
func releaseStowedItems(world w.World, cube ecs.Entity) {
	cargo := query.StowedCargo(world, cube)
	base := world.Components.GridElement.Get(cube).Coord
	for _, item := range cargo {
		coord := base.Add(world.Components.LocationStowed.Get(item).Offset)
		MoveMembersToField(world, []ecs.Entity{item}, coord, cube)
	}
}

// chebyshev は2タイル間のチェビシェフ距離を返す。斜め1マスも距離1として野営の広さに含める。
func chebyshev(a, b consts.Coord[consts.Tile]) int {
	dx := int(a.X - b.X)
	if dx < 0 {
		dx = -dx
	}
	dy := int(a.Y - b.Y)
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}
