package overworld

import (
	"fmt"
	"maps"
	"slices"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// 地上の地図の生成の途中経過。マップ生成の PlannerChain が段階ごとに MetaPlan の写しを取るのと同じく、地上の地図の生成の
// 段階を終えるたびに、その時点の地図の写しを記録係 mapRecorder に残す。段階は記録係の place か snapshot を呼ぶだけで
// 途中経過に加わり、MapSnapshots が写しを俯瞰図と同じ描画モデルにして返す

// mapRecorder は地上の地図の生成の段階ごとの写しを残す記録係。nil の記録係は何も残さず、写しの名前も作らないので、本番の生成は
// nil を渡す。ゼロ値のまま使える
type mapRecorder struct {
	// placed はここまでの段階で置いたチャンクの種類。写しは置いた種類のチャンクだけを描く
	placed    map[chunkType]bool
	snapshots []mapSnapshot
}

// mapSnapshot は地上の地図の生成の1段階を終えた時点の写し
type mapSnapshot struct {
	label       string
	placed      map[chunkType]bool
	roads       map[consts.Coord[consts.Chunk]][]roadPort
	settlements map[consts.Coord[consts.Chunk]]RoadDir
	reserved    map[consts.Coord[consts.Chunk]]bool
}

// place はチャンクの種類 kind を置いた段階として、label の名前で写しを残す
func (r *mapRecorder) place(label string, plan *roadPlan, kind chunkType) {
	if r == nil {
		return
	}
	if r.placed == nil {
		r.placed = map[chunkType]bool{}
	}
	r.placed[kind] = true
	r.snapshot(label, plan)
}

// road は経路を step 本目まで足した段階として写しを残す
func (r *mapRecorder) road(step int, plan *roadPlan) {
	if r == nil {
		return
	}
	r.snapshot(fmt.Sprintf("road %d", step), plan)
}

// snapshot は今の地図の写しを label の名前で残す
func (r *mapRecorder) snapshot(label string, plan *roadPlan) {
	if r == nil {
		return
	}
	roads := make(map[consts.Coord[consts.Chunk]][]roadPort, len(plan.roads))
	for c, ports := range plan.roads {
		roads[c] = slices.Clone(ports)
	}
	r.snapshots = append(r.snapshots, mapSnapshot{
		label:       label,
		placed:      maps.Clone(r.placed),
		roads:       roads,
		settlements: maps.Clone(plan.settlements),
		reserved:    maps.Clone(plan.reserved),
	})
}

// MapSnapshot は地上の地図の生成の途中の1段階を、俯瞰図と同じ描画モデルで表す
type MapSnapshot struct {
	Label string
	View  MacroView
}

// MapSnapshots は行 y を持つ計画区間の地上の地図を記録しながら作り直し、段階ごとの写しを北から南への俯瞰図にして返す。
// 段階は墜落現場、区域、行き先の予約、道を1本ずつ、集落、点在ランドマークの順に並ぶ。チャンクの種類は完成した地図から判定し、
// 段階の写しで、その段階までに置いた所だけを見せる
func MapSnapshots(raws rawschema.Raws, runSeed uint64, y, cols, spawnRow consts.Chunk) []MapSnapshot {
	road, ok := roadRow(raws)
	if !ok {
		return nil
	}
	rows := consts.Chunk(road.SpanRows)
	k := consts.FloorDiv(y, rows)
	rec := &mapRecorder{}
	plan := planSpan(raws, runSeed, road, k, cols, spawnRow, rec)
	rec.place("landmarks", plan, chunkLandmark)

	colorOf := glyphColorMap(raws)
	cat := zoneCatalogFrom(raws)
	wasteland := chunkGlyph(raws, rawschema.Wasteland)
	out := make([]MapSnapshot, 0, len(rec.snapshots))
	for _, s := range rec.snapshots {
		cells := make([][]MacroCell, rows)
		for cy := range rows {
			cells[cy] = make([]MacroCell, cols)
			for x := range cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: k*rows + cy}
				glyph := wasteland
				switch kind := chunkTypeAt(raws, runSeed, c, cols, spawnRow); {
				case kind == chunkSettlement && s.settlements[c] != 0:
					if g, ok := glyphByID(raws, settlementAt(raws, runSeed, c).Id); ok {
						glyph = g.Label
					}
				case kind != chunkSettlement && kind != chunkRoad && s.placed[kind]:
					glyph = chunkPlace(raws, cat, runSeed, c, cols, spawnRow)
				case s.reserved[c] && s.roads[c] == nil:
					glyph = reservedGlyph
				}
				col, hasCol := colorOf[glyph]
				cells[cy][x] = MacroCell{Glyph: glyph, Color: col, HasColor: hasCol, Discovered: true, Road: portDirs(s.roads[c])}
			}
		}
		out = append(out, MapSnapshot{Label: s.label, View: MacroView{Cells: cells}})
	}
	return out
}

// reservedGlyph は途中経過の俯瞰図で、行き先の前に予約したチャンクを示す記号
const reservedGlyph = '+'
