package overworld

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 道は隣接リージョンの小集落どうしを結ぶ舗装路。
// 集落の位置は Placement.WinnerOf で生成せずに算出できるため、各チャンクは自分を
// 横切る区間だけを独立に描ける。経路は西の集落から東の集落への L 字で、端点が
// リージョン順に正規化されているので、どちらのチャンクから生成しても同じ道になる。
type roadFeature struct{}

func (roadFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	// 区域の中は区域が街路を敷き、墜落現場の中は絵のとおりに残すので、道を描かない
	if kind := chunkTypeAt(world.Resources.RawMaster, runSeed, c, cols, spawnRowOf(world)); kind == chunkUrban || kind == chunkCrashSite {
		return nil
	}
	tiles := g.tiles.get()
	for _, link := range crossingRoads(world.Resources.RawMaster, runSeed, c, cols) {
		segs := roadPath(world.Resources.RawMaster, runSeed, link.a, link.b, cols, spawnRowOf(world), g.chunkW, g.chunkH)
		if err := drawRoadSegments(world, tiles, segs, link.road, c, g); err != nil {
			return err
		}
	}
	return nil
}

// drawRoadSegments は道の辺 segs のうち、チャンク c に含まれるマスだけを道 road のタイルで舗装する。既存タイルが土のマスだけを
// 置き換え、市街地の壁や床、集落は壊さない。幅は中心線に対し進行方向と垂直へ広げ、偶数なら中心からわずかに片側へ寄せる
func drawRoadSegments(world w.World, tiles map[gc.GridElement]ecs.Entity, segs []roadSeg, road rawschema.Road, c consts.Coord[consts.Chunk], g chunkGeom) error {
	earth := earthTilesOf(world.Resources.RawMaster)
	pave := func(px, py consts.Tile) error {
		loX := c.X.Tiles(g.chunkW)
		loY := c.Y.Tiles(g.chunkH)
		if px < loX || px >= loX+g.chunkW || py < loY || py >= loY+g.chunkH {
			return nil
		}
		wx := g.offsetX + (px - loX)
		wy := g.offsetY + (py - loY)
		if err := paveEarthTile(world, tiles, earth, consts.Coord[consts.Tile]{X: wx, Y: wy}, road.Tile); err != nil {
			return fmt.Errorf("failed to pave road (x=%d, y=%d): %w", wx, wy, err)
		}
		return nil
	}

	// 各辺を道の幅のバンドで敷く。進行軸に沿って可変軸を進め、垂直に幅ぶん広げる。
	// 角付近は両辺のバンドが重なるが paveEarthTile は冪等なので二重舗装は無害。
	width := consts.Tile(road.Width)
	for _, seg := range segs {
		for v := seg.lo; v <= seg.hi; v++ {
			for w := range width {
				off := w - width/2
				px, py := v, seg.fixed+off
				if seg.orient == orientVertical {
					px, py = seg.fixed+off, v
				}
				if err := pave(px, py); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// paveEarthTile は座標のタイルが土の地面のときだけ道のタイル tile へ置き換える。土でないタイルは他の地物の
// 生成物なので触らない。
func paveEarthTile(world w.World, tiles map[gc.GridElement]ecs.Entity, earth earthTiles, pos consts.Coord[consts.Tile], tile string) error {
	if !earth.at(world, tiles, pos) {
		return nil
	}
	_, err := replaceTile(world, tiles, pos, tile)
	return err
}
