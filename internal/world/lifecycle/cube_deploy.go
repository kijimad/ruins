package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

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

// DeployCube はキューブを展開状態にする。野営 CubeDeployCampRadius 内が開けていれば Deployed を付け、
// 畳み込んでいた貨物を元の相対位置へ出し直して true を返す。1タイルでも壁や敵、物で塞がれていれば
// 状態を変えず false を返す。プレイヤーは展開前に草などを壊して野営分の場所を空ける。
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

// deploySpaceFree は野営 CubeDeployCampRadius 内が展開に使えるかを返す。壁・敵・フィールドのアイテムや
// prop が1つでもあれば偽。判定範囲を圧縮時の畳み込み範囲と同じにすることで、展開できたら野営全体は空だと
// 保証され、圧縮で畳む物はすべてプレイヤーが後から置いた物になる。メニューは隣接で開くのでプレイヤーは
// 野営内に立つ。展開を妨げないよう本人は除外する。
func deploySpaceFree(world w.World, cube ecs.Entity) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}
	// プレイヤーは野営内に立ってメニューを開くので、本人のタイルで展開を拒否しないよう除外する。
	// 取得できないときは player が InvalidEntity になるが、CharacterAt が返す実キャラは決して
	// InvalidEntity と一致しないため、誰も除外しないだけで安全側に倒れる。
	player, _ := query.GetPlayerEntity(world)
	base := world.Components.GridElement.Get(cube).Coord
	for dy := -consts.CubeDeployCampRadius; dy <= consts.CubeDeployCampRadius; dy++ {
		for dx := -consts.CubeDeployCampRadius; dx <= consts.CubeDeployCampRadius; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			t := base.Add(consts.Coord[consts.Tile]{X: consts.Tile(dx), Y: consts.Tile(dy)})
			if si.IsBlockPass(t) {
				return false
			}
			if e, ok := si.CharacterAt(t); ok && e != player {
				return false
			}
		}
	}
	// フィールドのアイテムや prop が野営内にあれば展開しない。空間索引は BlockPass と character しか
	// 持たないので LocationOnField を走査する。早期 return でもロックを残さないよう defer で Close する
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	defer q.Close()
	for q.Next() {
		e := q.Entity()
		if e == cube || !world.Components.GridElement.Has(e) {
			continue
		}
		if chebyshev(world.Components.GridElement.Get(e).Coord, base) <= consts.CubeDeployCampRadius {
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

// stowNearbyItems は野営 CubeDeployCampRadius 内のフィールドのアイテムと prop をキューブへ畳み込む。
// LocationStowed へ移して燃料と区別し、キューブからの相対位置を覚えて展開で同じ配置へ戻せるようにする。
// 展開時に野営内は空だと保証されるので、ここにあるのはプレイヤーが後から置いた物だけで、自然の草・木を
// 吸い込む心配はない。プレイヤーの所持と同じく積める重量に硬い上限はなく、積みすぎれば運転の燃費が上がり
// 燃料が尽きて動けなくなるだけ。キャラクターは LocationOnField を持たず対象外、キューブ自身も除外する。
func stowNearbyItems(world w.World, cube ecs.Entity) {
	base := world.Components.GridElement.Get(cube).Coord

	var targets []ecs.Entity
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	defer q.Close()
	for q.Next() {
		e := q.Entity()
		if e == cube || !world.Components.GridElement.Has(e) {
			continue
		}
		if !world.Components.Item.Has(e) && !world.Components.Prop.Has(e) {
			continue
		}
		if chebyshev(world.Components.GridElement.Get(e).Coord, base) <= consts.CubeDeployCampRadius {
			targets = append(targets, e)
		}
	}
	for _, e := range targets {
		offset := world.Components.GridElement.Get(e).Sub(base)
		MoveToStowed(world, e, cube, offset)
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
