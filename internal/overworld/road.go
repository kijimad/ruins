package overworld

import (
	"fmt"
	"iter"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 道は区域・集落・墜落現場をつなぐ舗装路。計画区間の計画 road_plan.go が道のチャンクと出入り口を決め、
// 各チャンクは自分の出入り口だけを舗装する。
type roadFeature struct{}

// place は道のチャンクに、各出入り口から中心まで道を舗装する
func (roadFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	segs, road, ok, err := roadSegsAt(world.Resources.RawMaster, runSeed, c, cols, spawnRowOf(world), g.chunkW, g.chunkH)
	if err != nil || !ok {
		return err
	}
	return drawRoadSegments(world, g.tiles.get(), segs, road, c, g)
}

// drawRoadSegments は道の辺 segs のうち、チャンク c に含まれるマスだけを道 road のタイルで舗装する。既存タイルが土のマスだけを
// 置き換える。幅は中心線に対し進行方向と垂直へ広げ、偶数なら中心からわずかに片側へ寄せる
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

	// 角付近は両辺のバンドが重なるが paveEarthTile は冪等なので二重舗装は無害
	for p := range roadBand(segs, road.Width) {
		if err := pave(p.X, p.Y); err != nil {
			return err
		}
	}
	return nil
}

// roadBand は道の辺 segs を幅 width のバンドに広げたマスを、絶対タイル座標で順に返す。進行軸に沿って可変軸を進め、
// 垂直に幅ぶん広げる
func roadBand(segs []roadSeg, width int) iter.Seq[consts.Coord[consts.Tile]] {
	return func(yield func(consts.Coord[consts.Tile]) bool) {
		wd := consts.Tile(width)
		for _, seg := range segs {
			for v := seg.lo; v <= seg.hi; v++ {
				for w := range wd {
					off := w - wd/2
					p := consts.Coord[consts.Tile]{X: v, Y: seg.fixed + off}
					if seg.orient == orientVertical {
						p = consts.Coord[consts.Tile]{X: seg.fixed + off, Y: v}
					}
					if !yield(p) {
						return
					}
				}
			}
		}
	}
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
