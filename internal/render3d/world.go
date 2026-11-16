package render3d

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ErrNoPlayerCamera はプレイヤーのカメラが無いときに返す
var ErrNoPlayerCamera = errors.New("player camera not found")

// WorldProjector は world の状態から投影を組む。画面寸法もカメラ姿勢も world から読み、呼び出し側に取らせない。
//
// ダンジョン描画はプレイヤーの存在を前提とする。カメラやプレイヤーが無ければ投影は組めないので、
// error を返して呼び出し側の描画をやめさせる。
func WorldProjector(world w.World) (Projector, error) {
	sw, sh := world.Resources.GetScreenDimensions()
	camera := query.GetPlayerCamera(world)
	if camera == nil {
		return Projector{}, ErrNoPlayerCamera
	}
	tile, err := PlayerTile(world)
	if err != nil {
		return Projector{}, err
	}
	return NewProjector(*camera, tile, sw, sh), nil
}

// PlayerTile はプレイヤーの立つタイルを返す。投影の注視点と、描画範囲を絞るカリングの中心が
// 同じ位置を指すようにする。プレイヤーや位置が無ければ error を返す。
func PlayerTile(world w.World) (consts.Coord[consts.Tile], error) {
	pe, err := query.GetPlayerEntity(world)
	if err != nil {
		return consts.Coord[consts.Tile]{}, err
	}
	if !world.Components.GridElement.Has(pe) {
		return consts.Coord[consts.Tile]{}, fmt.Errorf("player has no GridElement")
	}
	return world.Components.GridElement.Get(pe).Coord, nil
}

// IsBox はその座標に高さのある箱として描かれるものがあるかを返す。
//
// 箱になるのは、描画順が高さのあるもの DepthNumTaller のタイルと、壁にはめる形 inset の
// 閉じた置物である。キューブのように通行を塞ぐだけの物は箱にならないので、通れるかで代わりに
// 判定してはいけない。
//
// 内部でエンティティ検索のクエリを開くので、クエリ反復中には呼ばない。呼び出し元は Draw の
// カーソル描画で、その時点でアクティブなクエリは無い。
func IsBox(world w.World, c consts.Coord[consts.Tile]) bool {
	q := query.ActiveFilter2[gc.GridElement, gc.SpriteRender](world).Query()
	for q.Next() {
		if IsBoxEntity(world, q.Entity()) && world.Components.GridElement.Get(q.Entity()).Coord == c {
			q.Close()
			return true
		}
	}
	return false
}

// IsBoxEntity は箱として描かれるものの述語。箱の集合を組む側と、タイルを指す側が
// 同じ判定を通ることで、どれが箱かの認識が2箇所に分かれない。
// Has を見るだけでクエリを開かないので、BoxSet のクエリ反復中から呼んでも安全。
// 座標からエンティティを引く IsBox とはこの点が異なる。
// 扉は開くと通行を塞がなくなるので、閉じている間だけ箱になる
func IsBoxEntity(world w.World, e ecs.Entity) bool {
	if !world.Components.SpriteRender.Has(e) {
		return false
	}
	sr := world.Components.SpriteRender.Get(e)
	if world.Components.Tile.Has(e) {
		return sr.Depth == gc.DepthNumTaller
	}
	return sr.Shape == rawschema.Inset && world.Components.BlockPass.Has(e)
}

// BoxSet は箱として描かれるものの座標の集合 boxes と、そのうちタイルの座標の集合 tiles を組む。
// boxes は箱の側面を隣と突き合わせて省くのに使う。tiles は壁を背にする置物の向きを決めるのに使い、
// 扉の開け閉めで置物の向きが変わらないよう置物の箱を含めない
func BoxSet(world w.World) (boxes, tiles map[consts.Coord[consts.Tile]]bool) {
	boxes = map[consts.Coord[consts.Tile]]bool{}
	tiles = map[consts.Coord[consts.Tile]]bool{}
	q := query.ActiveFilter2[gc.GridElement, gc.SpriteRender](world).Query()
	for q.Next() {
		e := q.Entity()
		if !IsBoxEntity(world, e) {
			continue
		}
		c := world.Components.GridElement.Get(e).Coord
		boxes[c] = true
		if world.Components.Tile.Has(e) {
			tiles[c] = true
		}
	}
	return boxes, tiles
}

// TileTopHeight はタイル上面の高さを返す。壁は箱として描かれるので天面、床は地面になる。
// タイルを指すカーソルはこの高さへ描く。壁タイルを地面の高さで描くと箱に埋もれて見えなくなる。
func TileTopHeight(world w.World, c consts.Coord[consts.Tile]) float64 {
	if IsBox(world, c) {
		return WallHeight
	}
	return 0
}
