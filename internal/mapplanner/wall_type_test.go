package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

func TestPlanData_GetWallType(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成（7x7）
	width, height := consts.Tile(7), consts.Tile(7)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]oapi.Tile, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]gc.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// テストケース1: WallTypeTop（下に床がある壁）
	// 座標系注意: XYTileIndex(tx Row, ty Col) → tx は X座標（横方向）、ty は Y座標（縦方向）
	// インデックス計算: ty * width + tx
	centerWallX, centerWallY := consts.Tile(3), consts.Tile(3)
	bottomFloorX, bottomFloorY := centerWallX, centerWallY+1 // 下の床（Y座標が大きくなる）

	centerWallIdx := planData.Level.XYTileIndex(centerWallX, centerWallY)
	bottomFloorIdx := planData.Level.XYTileIndex(bottomFloorX, bottomFloorY)

	planData.Tiles[bottomFloorIdx] = planData.GetTile("floor")

	// デバッグ情報を追加
	upFloor := planData.isFloorOrWarp(planData.UpTile(centerWallIdx))
	downFloor := planData.isFloorOrWarp(planData.DownTile(centerWallIdx))
	leftFloor := planData.isFloorOrWarp(planData.LeftTile(centerWallIdx))
	rightFloor := planData.isFloorOrWarp(planData.RightTile(centerWallIdx))

	wallType := planData.GetWallType(centerWallIdx)
	assert.Equal(t, WallTypeTop, wallType,
		"WallTypeTopの判定が間違っています。上:%t, 下:%t, 左:%t, 右:%t", upFloor, downFloor, leftFloor, rightFloor)

	// テストケース2: WallTypeRight（左に床がある壁）
	leftFloorX, leftFloorY := centerWallX-1, centerWallY // 左の床（X座標が小さくなる）
	leftFloorIdx := planData.Level.XYTileIndex(leftFloorX, leftFloorY)

	planData.Tiles[leftFloorIdx] = planData.GetTile("floor")
	planData.Tiles[bottomFloorIdx] = planData.GetTile("wall") // 前のテストケースをリセット

	wallType = planData.GetWallType(centerWallIdx)
	assert.Equal(t, WallTypeRight, wallType, "WallTypeRightの判定が間違っています")

	// テストケース3: WallTypeTopLeft（右下に床がある角壁）
	rightFloorX, rightFloorY := centerWallX+1, centerWallY // 右の床（X座標が大きくなる）
	downFloorX, downFloorY := centerWallX, centerWallY+1   // 下の床（Y座標が大きくなる）

	rightFloorIdx := planData.Level.XYTileIndex(rightFloorX, rightFloorY)
	downFloorIdx := planData.Level.XYTileIndex(downFloorX, downFloorY)

	planData.Tiles[rightFloorIdx] = planData.GetTile("floor")
	planData.Tiles[downFloorIdx] = planData.GetTile("floor")
	planData.Tiles[leftFloorIdx] = planData.GetTile("wall") // リセット

	wallType = planData.GetWallType(centerWallIdx)
	assert.Equal(t, WallTypeTopLeft, wallType, "WallTypeTopLeftの判定が間違っています")

	// テストケース4: WallTypeGeneric（複雑なパターン）
	upFloorX, upFloorY := centerWallX, centerWallY-1 // 上の床（Y座標が小さくなる）
	upFloorIdx := planData.Level.XYTileIndex(upFloorX, upFloorY)
	planData.Tiles[upFloorIdx] = planData.GetTile("floor")

	wallType = planData.GetWallType(centerWallIdx) // 今は上、右、下に床がある状態
	assert.Equal(t, WallTypeGeneric, wallType, "WallTypeGenericの判定が間違っています")
}

func TestPlanData_GetWallType_WithWarpTiles(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成
	width, height := consts.Tile(5), consts.Tile(5)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]oapi.Tile, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]gc.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 壁と床を配置
	wallX, wallY := consts.Tile(2), consts.Tile(2)
	floorX, floorY := wallX, wallY+1 // 下に床を配置（Y座標が大きくなる）

	floorIdx := planData.Level.XYTileIndex(floorX, floorY)
	wallIdx := planData.Level.XYTileIndex(wallX, wallY)
	planData.Tiles[floorIdx] = planData.GetTile("floor")

	wallType := planData.GetWallType(wallIdx)
	assert.Equal(t, WallTypeTop, wallType, "床タイルに対するWallTypeTopの判定が間違っています")
}
