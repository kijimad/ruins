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

// StowDefaultCubeCargo はキューブに既定の貨物を1つ畳み込む。ゲーム開始時のキューブ生成でだけ呼ぶ。
func StowDefaultCubeCargo(world w.World, cube ecs.Entity) error {
	item, err := spawnItemBase(world, defaultCubeCargoItem)
	if err != nil {
		return err
	}
	MoveToStowed(world, item, cube, defaultCubeCargoOffset)
	return nil
}

// DeployCube はキューブを展開する。野営 CubeDeployCampRadius 内が開いていれば Deployed を付けて
// 貨物を相対位置へ出し直し true、1タイルでも塞がれていれば状態を変えず false を返す。
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

// deploySpaceFree は野営 CubeDeployCampRadius 内が展開に使えるかを返す。壁・敵・アイテム・prop が
// 1つでもあれば偽。判定範囲を畳み込み範囲と同じにするので、展開できたら野営は空だと保証される。
func deploySpaceFree(world w.World, cube ecs.Entity) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}
	// プレイヤーは野営内に立ってメニューを開くので本人のタイルでは拒否しない。取得失敗時は
	// InvalidEntity になるが実キャラと一致しないので、誰も除外しないだけで安全側に倒れる。
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
	// 空間索引は BlockPass と character しか持たないのでアイテム・prop は LocationOnField を走査する
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

// stowNearbyItems は野営 CubeDeployCampRadius 内のアイテムと prop を LocationStowed へ畳み込み、燃料と
// 区別しつつ相対位置を覚えて展開で戻せるようにする。展開時に野営は空なので、畳むのはプレイヤーが置いた物だけ。
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

// releaseStowedItems は畳み込んだ貨物を記録した相対位置へ出し直す。キューブが移動していても相対位置で戻す。
func releaseStowedItems(world w.World, cube ecs.Entity) {
	cargo := query.StowedCargo(world, cube)
	base := world.Components.GridElement.Get(cube).Coord
	// 貨物ごとに別タイルへ戻すので MoveMembersToField を1件ずつ呼ぶ。スライスを使い回して確保を避ける
	single := make([]ecs.Entity, 1)
	for _, item := range cargo {
		single[0] = item
		coord := base.Add(world.Components.LocationStowed.Get(item).Offset)
		MoveMembersToField(world, single, coord, cube)
	}
}

// chebyshev は2タイル間のチェビシェフ距離を返す。斜めも距離1に含める。
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
