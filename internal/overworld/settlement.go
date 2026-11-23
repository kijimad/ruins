package overworld

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

// settlementFeature は集落の feature 実装。道の脇の集落のチャンクの中央に、集落の行が指す施設の絵を1つ置く。
// 施設は扉が道の側を向く向きで置き、チャンクの縁から施設の玄関まで道を伸ばす
type settlementFeature struct{}

func (settlementFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	raws := world.Resources.RawMaster
	if chunkTypeAt(raws, runSeed, c, cols, spawnRowOf(world)) != chunkSettlement {
		return nil
	}
	l, err := settlementLayoutAt(raws, runSeed, c, cols, spawnRowOf(world), g.chunkW, g.chunkH)
	if err != nil {
		return err
	}
	if _, _, err := spawnFacility(world, g, l.cells, l.origin.X, l.origin.Y, l.fac, urbanDangerAt(world, c), l.seed); err != nil {
		return fmt.Errorf("failed to place settlement %q: %w", l.fac, err)
	}
	road, ok := roadRow(raws)
	if !ok {
		return nil
	}
	return drawRoadSegments(world, g.tiles.get(), l.drivewaySegs(c, g.chunkW, g.chunkH), road, c, g)
}

// settlementLayout は集落のチャンクに置く施設の絵と向き
type settlementLayout struct {
	fac    string
	seed   uint64
	cells  [][]maptemplate.MapCell
	origin consts.Coord[consts.Tile] // 絵の左上のチャンクの中の座標
	facing RoadDir                   // 道のある側
	// entrance は道のある側の縁に沿った玄関の位置。チャンクの中の座標
	entrance consts.Tile
}

// settlementLayoutAt は集落 c の施設の絵を引き、扉が道の側を向く向きで中央に置いた形を返す。集落の生成と、集落へ向かう道の両方が
// これを引くので、道は玄関に着く
func settlementLayoutAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk, chunkW, chunkH consts.Tile) (settlementLayout, error) {
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return settlementLayout{}, fmt.Errorf("failed to load facility templates: %w", err)
	}
	fac := settlementAt(raws, runSeed, c).Facility
	seed := consts.Mix2D(runSeed^placementOf(raws, rawschema.PointFeatureKindSettlement).Salt, c.X, c.Y)
	// seed は settlementAt が種類を選ぶ値と同じだが、ストリーム 0 で絵の抽選に分ける。spawnFacility は戦利品と敵に 0x4 と 0x5 を使う
	rng := rand.New(rand.NewPCG(seed, 0))
	_, base, err := loader.FacilityChunk(fac, rng.Uint64())
	if err != nil {
		return settlementLayout{}, fmt.Errorf("failed to plan settlement %q: %w", fac, err)
	}
	facing := settlementFacing(raws, runSeed, c, cols, spawnRow)
	cells := facingCells(base, facing, rng)
	origin := consts.Coord[consts.Tile]{X: chunkW/2 - consts.Tile(len(cells[0]))/2, Y: chunkH/2 - consts.Tile(len(cells))/2}
	along := origin.X
	if facing == RoadW || facing == RoadE {
		along = origin.Y
	}
	return settlementLayout{fac: fac, seed: seed, cells: cells, origin: origin, facing: facing, entrance: along + entranceOf(cells, facing)}, nil
}

// drivewaySegs はチャンクの縁から施設の絵の手前まで、玄関の位置を通る道の辺を絶対タイル座標で返す。絵の中は施設の柵や地面のままにする
func (l settlementLayout) drivewaySegs(c consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile) []roadSeg {
	x0, y0 := c.X.Tiles(chunkW), c.Y.Tiles(chunkH)
	w, h := consts.Tile(len(l.cells[0])), consts.Tile(len(l.cells))
	var seg roadSeg
	switch l.facing {
	case RoadN:
		seg = roadSeg{orient: orientVertical, fixed: x0 + l.entrance, lo: y0, hi: y0 + l.origin.Y - 1}
	case RoadS:
		seg = roadSeg{orient: orientVertical, fixed: x0 + l.entrance, lo: y0 + l.origin.Y + h, hi: y0 + chunkH - 1}
	case RoadW:
		seg = roadSeg{orient: orientHorizontal, fixed: y0 + l.entrance, lo: x0, hi: x0 + l.origin.X - 1}
	case RoadE:
		seg = roadSeg{orient: orientHorizontal, fixed: y0 + l.entrance, lo: x0 + l.origin.X + w, hi: x0 + chunkW - 1}
	default:
		return nil
	}
	if seg.lo > seg.hi {
		return nil
	}
	return []roadSeg{seg}
}

// entranceOf は絵 cells の side の側の縁に沿った玄関の位置を、絵の中の座標で返す。玄関はその縁の両側を柵に挟まれた切れ目の真ん中にする。
// 切れ目が無ければ、その側に最も近い扉の並びの真ん中にする。偶数の幅の道が切れ目の真ん中に来るよう、真ん中は大きい側へ寄せる
func entranceOf(cells [][]maptemplate.MapCell, side RoadDir) consts.Tile {
	middle := func(r [2]int) consts.Tile { return consts.Tile((r[0] + r[1] + 1) / 2) }
	if r, ok := fenceGap(cells, side); ok {
		return middle(r)
	}
	depthMax := len(cells)
	if side == RoadW || side == RoadE {
		depthMax = len(cells[0])
	}
	for depth := range depthMax {
		if doors := runsOf(edgeLine(cells, side, depth), func(c maptemplate.MapCell) bool { return c.Prop == "door" }); len(doors) > 0 {
			return middle(doors[0])
		}
	}
	return consts.Tile(len(edgeLine(cells, side, 0)) / 2)
}

// entranceSide は絵の玄関がある側を返す。縁に柵がある絵では、両側を柵に挟まれた切れ目がある縁の側にする。柵の無い絵では、
// 縁に最も近い扉の側にする。玄関の側が決まらなければ0
func entranceSide(cells [][]maptemplate.MapCell) RoadDir {
	fenced := false
	for _, d := range roadDirs {
		if _, ok := fenceGap(cells, d); ok {
			return d
		}
		fenced = fenced || slices.ContainsFunc(edgeLine(cells, d, 0), isFence)
	}
	if fenced {
		return 0
	}
	return doorSide(cells)
}

// fenceGap は絵の side の側の縁で、両側を柵に挟まれた切れ目の範囲を返す。縁の角の柵の無いマスは切れ目にしない
func fenceGap(cells [][]maptemplate.MapCell, side RoadDir) ([2]int, bool) {
	edge := edgeLine(cells, side, 0)
	for _, r := range runsOf(edge, func(c maptemplate.MapCell) bool { return !isFence(c) }) {
		if r[0] > 0 && r[1] < len(edge)-1 {
			return r, true
		}
	}
	return [2]int{}, false
}

// edgeLine は絵の side の側の縁から depth だけ内側の線のマスを、縁に沿って並べて返す
func edgeLine(cells [][]maptemplate.MapCell, side RoadDir, depth int) []maptemplate.MapCell {
	h, w := len(cells), len(cells[0])
	switch side {
	case RoadS:
		return slices.Clone(cells[h-1-depth])
	case RoadW, RoadE:
		x := depth
		if side == RoadE {
			x = w - 1 - depth
		}
		line := make([]maptemplate.MapCell, h)
		for y := range h {
			line[y] = cells[y][x]
		}
		return line
	default:
		return slices.Clone(cells[depth])
	}
}

// runsOf は line の中で match するマスの連続した範囲を、両端を含む添字の組で並べる
func runsOf(line []maptemplate.MapCell, match func(maptemplate.MapCell) bool) [][2]int {
	var out [][2]int
	for i, c := range line {
		if !match(c) {
			continue
		}
		if n := len(out); n > 0 && out[n-1][1] == i-1 {
			out[n-1][1] = i
		} else {
			out = append(out, [2]int{i, i})
		}
	}
	return out
}

// isFence はマスに柵が立つかを返す
func isFence(c maptemplate.MapCell) bool { return c.Prop == "fence" }

// settlementAt は集落の種類を raw の settlements の行から重みの比で選ぶ。純粋に (runSeed, 座標) の
// 関数で、地図の分類と生成の両方がこれを使うので種類の見た目と実体が一致する。
// 行は読み込み時に1つ以上あり、重みは1以上であることを検証している。
func settlementAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk]) rawschema.Settlement {
	rows := raw.PtrSlice(raws.Settlements)
	total := 0
	for i := range rows {
		total += int(rows[i].Weight)
	}
	roll := int(consts.Mix2D(runSeed^placementOf(raws, rawschema.PointFeatureKindSettlement).Salt, c.X, c.Y) % uint64(total))
	for i := range rows {
		roll -= int(rows[i].Weight)
		if roll < 0 {
			return rows[i]
		}
	}
	return rows[len(rows)-1]
}

// facingCells は施設の絵 cells を、玄関が facing の側を向く回転と反転から rng で1つ選んで返す。玄関がその側を向く向きが無ければ、
// 全ての向きから選ぶ
func facingCells(cells [][]maptemplate.MapCell, facing RoadDir, rng *rand.Rand) [][]maptemplate.MapCell {
	all := make([][][]maptemplate.MapCell, 0, 8)
	var matched [][][]maptemplate.MapCell
	for turns := range 4 {
		for _, mirror := range []bool{false, true} {
			t := maptemplate.TransformCells(cells, turns, mirror)
			all = append(all, t)
			if entranceSide(t) == facing {
				matched = append(matched, t)
			}
		}
	}
	if len(matched) == 0 {
		matched = all
	}
	return matched[rng.IntN(len(matched))]
}

// doorSide は絵の扉のうち縁に最も近いものが向く側を返す。扉が無ければ0
func doorSide(cells [][]maptemplate.MapCell) RoadDir {
	h, w := len(cells), len(cells[0])
	best, side := h+w, RoadDir(0)
	for y, row := range cells {
		for x, cell := range row {
			if cell.Prop != "door" {
				continue
			}
			for _, e := range []struct {
				d    int
				side RoadDir
			}{{y, RoadN}, {h - 1 - y, RoadS}, {x, RoadW}, {w - 1 - x, RoadE}} {
				if e.d < best {
					best, side = e.d, e.side
				}
			}
		}
	}
	return side
}
