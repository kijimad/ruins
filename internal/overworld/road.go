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

// 道は隣接リージョンの小集落どうしを結ぶ舗装路。
// 集落の位置は Placement.WinnerOf で生成せずに算出できるため、各チャンクは自分を
// 横切る区間だけを独立に描ける。経路は西の集落から東の集落への L 字で、端点が
// リージョン順に正規化されているので、どちらのチャンクから生成しても同じ道になる。
// 道はチャンクの境目を通り、チャンクの中に置く施設や墜落現場の脇を過ぎる。
type roadFeature struct{}

func (roadFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	// 区域の中は区域が街路を敷くので、道を描かない
	if chunkTypeAt(world.Resources.RawMaster, runSeed, c, cols, spawnRowOf(world)) == chunkUrban {
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

// chunkRoadTiles はチャンク c を通る道が舗装するマスを、チャンク内の座標で返す。生成せずに道の経路から求める
func chunkRoadTiles(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk, chunkW, chunkH consts.Tile) map[consts.Coord[consts.Tile]]bool {
	lo := consts.Coord[consts.Tile]{X: c.X.Tiles(chunkW), Y: c.Y.Tiles(chunkH)}
	out := map[consts.Coord[consts.Tile]]bool{}
	for _, link := range crossingRoads(raws, runSeed, c, cols) {
		for p := range roadBand(roadPath(raws, runSeed, link.a, link.b, cols, spawnRow, chunkW, chunkH), link.road.Width) {
			if l := p.Sub(lo); l.X >= 0 && l.X < chunkW && l.Y >= 0 && l.Y < chunkH {
				out[l] = true
			}
		}
	}
	return out
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
