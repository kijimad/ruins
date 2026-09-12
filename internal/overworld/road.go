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

func (roadFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	tiles := g.tiles.get()
	for _, pair := range crossingRoads(runSeed, c, cols) {
		if err := drawRoadSegments(world, tiles, pair[0], pair[1], c, g); err != nil {
			return err
		}
	}
	return nil
}

// drawRoadSegments は集落 a の中心から b の中心への L 字経路のうち、チャンク c に
// 含まれるマスだけを舗装する。既存タイルが土のマスだけを置き換え、市街地の壁や床、
// 集落は壊さない。経路の分解は roadSegments を唯一の出典とし、地図・散布と一致させる。
func drawRoadSegments(world w.World, tiles map[gc.GridElement]ecs.Entity, a, b, c consts.Coord[consts.Chunk], g chunkGeom) error {
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

	// 各辺を幅 roadWidth のバンドで敷く。進行軸に沿って可変軸を進め、垂直に roadWidth ぶん広げる。
	// 角付近は両辺のバンドが重なるが replaceDirtTile は冪等なので二重舗装は無害。
	for _, seg := range roadSegments(a, b) {
		fixed, lo, hi := seg.tileSpan(g.chunkW, g.chunkH)
		for v := lo; v <= hi; v++ {
			for w := range roadWidth {
				off := w - roadWidth/2
				px, py := v, fixed+off
				if seg.orient == orientVertical {
					px, py = fixed+off, v
				}
				if err := pave(px, py); err != nil {
					return err
				}
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
	if !ok || !world.ECS.Alive(e) || !world.Components.RawID.Has(e) {
		return nil
	}
	if world.Components.RawID.Get(e).ID != consts.TileNameDirt {
		return nil
	}
	_, err := replaceTile(world, tiles, pos, consts.TileNameFloor)
	return err
}
