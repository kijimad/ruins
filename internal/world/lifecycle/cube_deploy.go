package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
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

// defaultCubeModuleItems はゲーム開始時にキューブへ積んでおく範囲モジュール。共通の拡張モジュールを2つ
// 積み、装着で展開範囲が段階的に伸びるのを序盤から試せるようにする。工作台での作成は将来。プレイヤーの
// バックパックでなくキューブ収納へ入れるので、所持重量は変わらず、装着はキューブメニューから行う。
var defaultCubeModuleItems = []string{"cube_range_module", "cube_range_module"}

// StockDefaultCubeModules はキューブ収納へ既定の範囲モジュールを積む。ゲーム開始時のキューブ生成でだけ呼ぶ。
func StockDefaultCubeModules(world w.World, cube ecs.Entity) error {
	for _, name := range defaultCubeModuleItems {
		item, err := spawnItemBase(world, name)
		if err != nil {
			return err
		}
		if err := MoveToStorage(world, item, cube); err != nil {
			return err
		}
	}
	return nil
}

// DeployCube はキューブを展開する。野営が開いていれば Deployed を付けて
// 貨物を相対位置へ出し直し true、1タイルでも塞がれていれば状態を変えず false を返す。
func DeployCube(world w.World, cube ecs.Entity) bool {
	if world.Components.Deployed.Has(cube) {
		return true
	}
	// 展開時のライブ範囲でフットプリントの空きを判定し、その範囲を凍結して持たせる。以後の畳み込みと
	// 壁描画は凍結値を使うので、展開中に装備で範囲が伸びても野営は変わらず既存 prop を巻き込まない。
	r := query.CubeDeployRange(world, cube)
	if !deploySpaceFree(world, cube, r) {
		return false
	}
	world.Components.Deployed.Add(cube, &gc.Deployed{Range: r})
	releaseStowedItems(world, cube)
	return true
}

// deploySpaceFree は野営内が展開に使えるかを返す。壁・敵・アイテム・prop が
// 1つでもあれば偽。判定範囲 r を畳み込み範囲と同じにするので、展開できたら野営は空だと保証される。
func deploySpaceFree(world w.World, cube ecs.Entity, r consts.Coord[consts.Tile]) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}
	// プレイヤーは野営内に立ってメニューを開くので本人のタイルでは拒否しない。取得失敗時は
	// InvalidEntity になるが実キャラと一致しないので、誰も除外しないだけで安全側に倒れる。
	player, _ := query.GetPlayerEntity(world)
	base := world.Components.GridElement.Get(cube).Coord
	for dy := -r.Y; dy <= r.Y; dy++ {
		for dx := -r.X; dx <= r.X; dx++ {
			// キューブ自身のタイルは障害物として見ない
			if dx == 0 && dy == 0 {
				continue
			}
			t := base.Add(consts.Coord[consts.Tile]{X: dx, Y: dy})
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
		if geometry.WithinRect(world.Components.GridElement.Get(e).Coord, base, r) {
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

// stowNearbyItems は野営内のアイテムと prop を LocationStowed へ畳み込み、燃料と
// 区別しつつ相対位置を覚えて展開で戻せるようにする。展開時に野営は空なので、畳むのはプレイヤーが置いた物だけ。
// 畳み込み範囲は展開時に凍結した Deployed.Range を使う。ライブ範囲だと展開後に伸びたぶん外周を巻き込む。
func stowNearbyItems(world w.World, cube ecs.Entity) {
	base := world.Components.GridElement.Get(cube).Coord
	r := world.Components.Deployed.Get(cube).Range

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
		if geometry.WithinRect(world.Components.GridElement.Get(e).Coord, base, r) {
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
