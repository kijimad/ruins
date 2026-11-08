package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// CubeTiles はキューブのタイル座標を集めて返す。マクロ地図のマーカー用。
// 位置は GridElement から常に導けるので保持せず、読み取り時に集める。
func CubeTiles(world w.World) []consts.Coord[consts.Tile] {
	var out []consts.Coord[consts.Tile]
	q := ActiveFilter2[gc.GridElement, gc.Cube](world).Query()
	for q.Next() {
		out = append(out, world.Components.GridElement.Get(q.Entity()).Coord)
	}
	return out
}

// DiscoveredChunks は探索済みタイルを含むチャンクの集合を絶対チャンク座標で返す。タイル単位の
// フォグ ExploredTiles をチャンク粒度へ畳み込み、マクロ地図のフォグにする。StageField が無ければ nil。
// 全画面の地形俯瞰図と HUD の右上地図が同じフォグを共有する。
func DiscoveredChunks(world w.World, sb *gc.SeamlessBand) map[consts.Coord[consts.Chunk]]bool {
	field := GetCurrentStageField(world)
	if field == nil {
		return nil
	}
	out := make(map[consts.Coord[consts.Chunk]]bool)
	// ChunkW は帯が有効なら正なのでゼロ除算しない。X は有界なので絶対チャンク列はタイルを幅で割るだけ。
	// Y は AbsChunkRow が帯ローカルの非負タイル行を絶対チャンク行へ移す。単一出どころを共有する
	for tile := range field.ExploredTiles {
		col := consts.Chunk(int(tile.X) / int(sb.ChunkW))
		out[consts.Coord[consts.Chunk]{X: col, Y: sb.AbsChunkRow(tile.Y)}] = true
	}
	return out
}

// PlayerBandTile はプレイヤーの帯ローカルなタイル座標を返す。プレイヤーが居なければ ok=false。
// マクロ地図のプレイヤーマーカーを置くのに使う。
func PlayerBandTile(world w.World) (consts.Coord[consts.Tile], bool) {
	player, err := GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return consts.Coord[consts.Tile]{}, false
	}
	return world.Components.GridElement.Get(player).Coord, true
}

// StowedCargo はキューブに畳み込んだ貨物の一覧を返す。LocationStowed で燃料タンクと別管理する。
// 反復中に return するとロックが残るので、対象を集めてから返す。
func StowedCargo(world w.World, cube ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	q := ecs.NewFilter1[gc.LocationStowed](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationStowed.Get(e).Owner == cube {
			items = append(items, e)
		}
	}
	return items
}

// GetCubeModules はキューブに装着したモジュールの一覧を返す。LocationInstalled で Owner を辿り、
// CubeModule も併せて絞るので、返る実体は性能コンポーネントを必ず持つ。読み取り側の Has チェックが要らない。
// 反復中に return するとロックが残るので、対象を集めてから返す。
func GetCubeModules(world w.World, cube ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	q := ecs.NewFilter2[gc.LocationInstalled, gc.CubeModule](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInstalled.Get(e).Owner == cube {
			items = append(items, e)
		}
	}
	return items
}

// BackpackCubeModules はプレイヤーのバックパックにあるキューブモジュールを返す。装着候補の一覧に使う。
// 反復中に return するとロックが残るので、対象を集めてから返す。
func BackpackCubeModules(world w.World, player ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	q := ecs.NewFilter2[gc.LocationInBackpack, gc.CubeModule](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInBackpack.Get(e).Owner == player {
			items = append(items, e)
		}
	}
	return items
}

// BackpackDeployables はプレイヤーのバックパックにある装着アイテムを返す。設備画面の装着候補に使う。
// 反復中に return するとロックが残るので、対象を集めてから返す。
func BackpackDeployables(world w.World, player ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	q := ecs.NewFilter2[gc.LocationInBackpack, gc.Deployable](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInBackpack.Get(e).Owner == player {
			items = append(items, e)
		}
	}
	return items
}

// FacilityAt は指定タイルのフィールドにある装着設備を返す。設備画面で撤去対象を引くのに使う。
// フィールドに装着されている Deployable だけを対象にするので、grass など装着でない prop は撤去できない。
// LocationOnField を明示して確認し、座標だけ残った別ロケーションの実体を誤って引かない。
func FacilityAt(world w.World, coord consts.Coord[consts.Tile]) (ecs.Entity, bool) {
	for _, e := range GetEntitiesAt(world, coord.X, coord.Y) {
		if world.Components.Deployable.Has(e) && world.Components.LocationOnField.Has(e) {
			return e, true
		}
	}
	return gc.InvalidEntity, false
}

// CubeDeployRange はキューブの展開野営の縦横別の半径を返す。基準 consts.CubeDeployBaseRange に、
// 装着した各モジュールの RangeBonus を縦横一律に加算する。範囲は保持せず読み取り時に導く。
// 展開判定・畳み込み・レーザー壁描画がこの単一出典を参照する。
func CubeDeployRange(world w.World, cube ecs.Entity) consts.Coord[consts.Tile] {
	r := consts.CubeDeployBaseRange()
	for _, m := range GetCubeModules(world, cube) {
		bonus := world.Components.CubeModule.Get(m).RangeBonus
		r.X += bonus
		r.Y += bonus
	}
	return r
}

// CubeFuelCapacity は基準容量に装着中モジュールの FuelCapacityBonus を足した燃料タンクの容量を返す。
func CubeFuelCapacity(world w.World, cube ecs.Entity) consts.Heat {
	c := consts.CubeFuelCapacityBase
	for _, m := range GetCubeModules(world, cube) {
		c += world.Components.CubeModule.Get(m).FuelCapacityBonus
	}
	return c
}

// CubeWeight はキューブが積む物の総重量を返す。運転1タイルの燃料コスト算出に使う。
// 燃料タンクの中身・畳み込んだ貨物・装着モジュールが機動に効くので合算する。値は保持せず読み取り時に導く。
func CubeWeight(world w.World, cube ecs.Entity) consts.Milligram {
	var total consts.Milligram
	for _, item := range GetStorageItems(world, cube) {
		total += GetEntityWeight(world, item)
	}
	for _, item := range StowedCargo(world, cube) {
		total += GetEntityWeight(world, item)
	}
	for _, item := range GetCubeModules(world, cube) {
		total += GetEntityWeight(world, item)
	}
	return total
}

// FollowFuelCost は総重量から1歩随伴するのに要する燃料ポイントを返す。空でも基準量がかかり、
// 総重量に比例して増える。重いほど燃費が悪化する。
func FollowFuelCost(total consts.Milligram) consts.Heat {
	kg := int(total / consts.MilligramPerKg)
	return consts.Heat(consts.FollowFuelBase + consts.FollowFuelPerKg*kg)
}

// CubeFuelTotal はキューブに貯めた燃料ポイントを返す。FuelTank を持たなければ0。
func CubeFuelTotal(world w.World, cube ecs.Entity) consts.Heat {
	if !world.Components.FuelTank.Has(cube) {
		return 0
	}
	return world.Components.FuelTank.Get(cube).Current
}

// FuelGaugeRatio は燃料ゲージの充填率を現在値と上限の比で返す。上限が0以下なら0。
func FuelGaugeRatio(tank gc.FuelTank) float64 {
	if tank.Max <= 0 {
		return 0
	}
	return min(1, max(0, float64(tank.Current)/float64(tank.Max)))
}

// ActiveCube は現ステージにいるキューブを返す。ダンジョンなどキューブのいないステージでは false。
func ActiveCube(world w.World) (ecs.Entity, bool) {
	q := ActiveFilter1[gc.Cube](world).Query()
	if !q.Next() {
		return gc.InvalidEntity, false
	}
	cube := q.Entity()
	q.Close()
	return cube, true
}
