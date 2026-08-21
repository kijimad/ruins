package render3d

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// defaultPlayerTile はプレイヤーがいないときに見る位置。マップ生成の確認など、
// プレイヤーを置かずに世界だけ描く場面で使う
var defaultPlayerTile = consts.Coord[consts.Tile]{X: 25, Y: 25}

// For は world の状態から投影を組む。画面寸法もカメラ姿勢も world から読み、呼び出し側には
// 取らせない。寸法やカメラの取り方が呼び出し側ごとに分かれると、投影が再び2系統へ割れるためである。
//
// カメラが無ければ既定の視点で組む。ここで投影を諦めるとカーソルが消えてしまい、
// 位置がずれるより分かりにくい壊れ方になる。
func For(world w.World) Projector {
	sw, sh := world.Resources.GetScreenDimensions()
	orbit := gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}
	if camera := query.GetPlayerCamera(world); camera != nil {
		orbit = *camera
	}
	return New(orbit, PlayerTile(world), sw, sh)
}

// PlayerTile はプレイヤーの立つタイルを返す。いなければ既定値を返す。
// 投影の注視点と、描画範囲を絞るカリングの中心が同じ位置を指すようにする
func PlayerTile(world w.World) consts.Coord[consts.Tile] {
	pe, err := query.GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(pe) {
		return defaultPlayerTile
	}
	return world.Components.GridElement.Get(pe).Coord
}

// IsWallTile はそのタイルが高さのある箱として描かれるかを返す。
//
// 箱になるのは Tile と BlockPass を併せ持つエンティティだけである。扉やキューブは通行を
// 塞ぐが箱にはならないので、SpatialIndex の通行不能判定をそのまま流用してはいけない。
// ただし通行できるタイルは箱にもならないので、索引で先に落として全走査を避ける。
//
// 内部でエンティティ検索のクエリを開くので、クエリ反復中には呼ばない。呼び出し元は Draw の
// カーソル描画で、その時点でアクティブなクエリは無い。
func IsWallTile(world w.World, c consts.Coord[consts.Tile]) bool {
	if si := query.GetSpatialIndex(world); si != nil && !si.IsBlockPass(c) {
		return false
	}
	for _, e := range query.GetEntitiesAt(world, c.X, c.Y) {
		if IsWallTileEntity(world, e) {
			return true
		}
	}
	return false
}

// IsWallTileEntity は箱として描かれるタイルの述語。壁の集合を組む側と、タイルを指す側が
// 同じ判定を通ることで、どのタイルが箱かの認識が2箇所に分かれない
func IsWallTileEntity(world w.World, e ecs.Entity) bool {
	return world.Components.Tile.Has(e) && world.Components.BlockPass.Has(e)
}

// WallTileSet は箱として描かれるタイルの集合を組む。壁の側面を隣と突き合わせて省くのに使う
func WallTileSet(world w.World) map[consts.Coord[consts.Tile]]bool {
	walls := map[consts.Coord[consts.Tile]]bool{}
	wallQ := query.ActiveFilter2[gc.GridElement, gc.BlockPass](world).With(ecs.C[gc.Tile]()).Query()
	for wallQ.Next() {
		walls[world.Components.GridElement.Get(wallQ.Entity()).Coord] = true
	}
	return walls
}

// TileTopHeight はタイル上面の高さを返す。壁は箱として描かれるので天面、床は地面になる。
// タイルを指すカーソルはこの高さへ描く。壁タイルを地面の高さで描くと箱に埋もれて見えなくなる。
func TileTopHeight(world w.World, c consts.Coord[consts.Tile]) float64 {
	if IsWallTile(world, c) {
		return WallHeight
	}
	return 0
}
