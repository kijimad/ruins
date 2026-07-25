package overworld

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
)

// 道は隣接リージョンの小集落どうしを結ぶ舗装路。Connection 層の v1。
// 集落の位置は Placement.WinnerOf で生成せずに算出できるため、各チャンクは自分を
// 横切る区間だけを独立に描ける。経路は西の集落から東の集落への L 字で、端点が
// リージョン順に正規化されているので、どちらのチャンクから生成しても同じ道になる。
type roadFeature struct{}

func (roadFeature) place(world w.World, runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error {
	r := floorDiv(c.X, settlementPlacement.Spacing)
	tiles := g.tiles.get()
	// c を横切りうるのは (r-1, r) と (r, r+1) を結ぶ2本だけ
	for _, pr := range []consts.Chunk{r - 1, r} {
		a := settlementPlacement.WinnerOf(runSeed, pr, rows)
		b := settlementPlacement.WinnerOf(runSeed, pr+1, rows)
		if err := drawRoadSegments(world, tiles, a, b, c, g); err != nil {
			return err
		}
	}
	return nil
}

// drawRoadSegments は集落 a の中心から b の中心への L 字経路のうち、チャンク c に
// 含まれるマスだけを舗装する。既存タイルが土のマスだけを置き換え、市街地の壁や床、
// 集落は壊さない。
func drawRoadSegments(world w.World, tiles map[gc.GridElement]ecs.Entity, a, b, c worldstream.ChunkCoord, g chunkGeom) error {
	ax := a.X.Tiles(g.chunkW) + g.chunkW/2
	ay := a.Y.Tiles(g.chunkH) + g.chunkH/2
	bx := b.X.Tiles(g.chunkW) + g.chunkW/2
	by := b.Y.Tiles(g.chunkH) + g.chunkH/2

	pave := func(px, py consts.Tile) error {
		loX := c.X.Tiles(g.chunkW)
		loY := c.Y.Tiles(g.chunkH)
		if px < loX || px >= loX+g.chunkW || py < loY || py >= loY+g.chunkH {
			return nil
		}
		wx := g.offsetX + (px - loX)
		wy := g.offsetY + (py - loY)
		if err := replaceDirtTile(world, tiles, wx, wy); err != nil {
			return fmt.Errorf("道の舗装に失敗 (x=%d, y=%d): %w", wx, wy, err)
		}
		return nil
	}

	for x := min(ax, bx); x <= max(ax, bx); x++ {
		if err := pave(x, ay); err != nil {
			return err
		}
	}
	for y := min(ay, by); y <= max(ay, by); y++ {
		if err := pave(bx, y); err != nil {
			return err
		}
	}
	return nil
}

// replaceDirtTile は座標のタイルが土のときだけ床へ置き換える。土以外は他の地物の
// 生成物なので触らない。
func replaceDirtTile(world w.World, tiles map[gc.GridElement]ecs.Entity, x, y consts.Tile) error {
	g := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
	e, ok := tiles[g]
	if !ok || !world.ECS.Alive(e) || !world.Components.Name.Has(e) {
		return nil
	}
	if world.Components.Name.Get(e).Name != consts.TileNameDirt {
		return nil
	}
	return replaceTile(world, tiles, x, y, consts.TileNameFloor)
}
