package overworld

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 道は隣接リージョンの小集落どうしを結ぶ舗装路。
// 集落の位置は Placement.WinnerOf で生成せずに算出できるため、各チャンクは自分を
// 横切る区間だけを独立に描ける。経路は西の集落から東の集落への L 字で、端点が
// リージョン順に正規化されているので、どちらのチャンクから生成しても同じ道になる。
type roadFeature struct{}

// roadWidth は舗装路の幅。中心線に対し進行方向と垂直へこのタイル数だけ広げる。1タイルだと
// 隊商の街道として細すぎるので幅を持たせる。幅は偶数なので中心は定まらず、オフセットを
// -roadWidth/2 から始めてわずかに片側へ寄せる。
const roadWidth consts.Tile = 4

func (roadFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error {
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
func drawRoadSegments(world w.World, tiles map[gc.GridElement]ecs.Entity, a, b, c consts.Coord[consts.Chunk], g chunkGeom) error {
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
		if err := replaceDirtTile(world, tiles, consts.Coord[consts.Tile]{X: wx, Y: wy}); err != nil {
			return fmt.Errorf("failed to pave road (x=%d, y=%d): %w", wx, wy, err)
		}
		return nil
	}

	// 水平辺は y=ay を中心に Y 方向へ、垂直辺は x=bx を中心に X 方向へ、幅 roadWidth のバンドで敷く。
	// 角の (bx, ay) 付近は両バンドが重なるが replaceDirtTile は冪等なので二重舗装は無害。
	for x := min(ax, bx); x <= max(ax, bx); x++ {
		for w := range roadWidth {
			if err := pave(x, ay+w-roadWidth/2); err != nil {
				return err
			}
		}
	}
	for y := min(ay, by); y <= max(ay, by); y++ {
		for w := range roadWidth {
			if err := pave(bx+w-roadWidth/2, y); err != nil {
				return err
			}
		}
	}
	return nil
}

// replaceDirtTile は座標のタイルが土のときだけ床へ置き換える。土以外は他の地物の
// 生成物なので触らない。
func replaceDirtTile(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile]) error {
	g := gc.GridElement{Coord: pos}
	e, ok := tiles[g]
	if !ok || !world.ECS.Alive(e) || !world.Components.Name.Has(e) {
		return nil
	}
	if world.Components.Name.Get(e).Name != consts.TileNameDirt {
		return nil
	}
	return replaceTile(world, tiles, pos, consts.TileNameFloor)
}
