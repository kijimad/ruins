package overworld

import (
	"cmp"
	"fmt"
	"image/color"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// チャンクマップ表記は、1チャンク=1建物という縮尺に合わせ、各チャンクを「そこが何の種別の
// 場所か」の1文字で表す俯瞰図。1チャンク=1記号で場所の種別を示す。生成は (runSeed, 座標) の
// 純関数なので、ECS のタイルを生成せず種別だけを算出できる。

// GlyphInfo は種別マップ上の1文字表記と、凡例に出す名前と色。記号・名前・色を1レコードに同梱し、
// UI 側が記号ごとに別の色表を持って二重定義でずれるのを防ぐ。UI の着色や凡例に使う。
type GlyphInfo struct {
	Label rune
	Name  string
	Color color.RGBA
}

// 俯瞰図の記号は尺度の違う2層の語彙でできている。両者は対等な兄弟ではなく、市街地以外の粗い層と
// 市街地の中の細かい層という入れ子の関係にある。地図はチャンクの種別に応じてどちらかの記号を1つ選ぶ。
//   - チャンク尺度。市街地以外のチャンクを1記号で表す。集落とランドマークは raw の行の id を記号の id に使い、
//     持ち主の行を持たない荒れ地と遺跡入口は、記号の行の chunk が指す。地図の表示専用で、生成には関与しない。
//   - 施設 id: 建物尺度。市街地チャンクの中の1建物の種別。住宅・商店・診療所など。表示だけでなく
//     市街地生成の重み抽選にも使う。
// 地図は市街地チャンクを施設 id の記号で、それ以外をチャンク尺度の記号で描く。凡例 LegendGlyphs は
// チャンク尺度に続けて建物尺度を並べ、2層を1つの表にする。

// placeUnknownGlyph は mapGlyphs に記号が無いときの保険の記号。
const placeUnknownGlyph rune = '?'

// toGlyphInfo は mapGlyphs 行を GlyphInfo へ変換する。
func toGlyphInfo(mg rawschema.MapGlyph) GlyphInfo {
	label, _ := utf8.DecodeRuneInString(mg.Glyph)
	return GlyphInfo{Label: label, Name: mg.Name, Color: color.RGBA{R: mg.Color.R, G: mg.Color.G, B: mg.Color.B, A: mg.Color.A}}
}

// glyphByID は種別 id から地図記号を引く。
func glyphByID(raws rawschema.Raws, id string) (GlyphInfo, bool) {
	mg, ok := raw.GetMapGlyph(raws, id)
	if !ok {
		return GlyphInfo{}, false
	}
	return toGlyphInfo(mg), true
}

// LegendGlyphs は俯瞰図の全記号を凡例順に返す。
func LegendGlyphs(raws rawschema.Raws) []GlyphInfo {
	mgs := raw.PtrSlice(raws.MapGlyphs)
	sorted := make([]rawschema.MapGlyph, len(mgs))
	copy(sorted, mgs)
	slices.SortStableFunc(sorted, func(a, b rawschema.MapGlyph) int { return cmp.Compare(a.Order, b.Order) })
	out := make([]GlyphInfo, 0, len(sorted))
	for _, mg := range sorted {
		out = append(out, toGlyphInfo(mg))
	}
	return out
}

// glyphColorMap は記号 rune から色への表を返す。
func glyphColorMap(raws rawschema.Raws) map[rune]color.RGBA {
	glyphs := LegendGlyphs(raws)
	table := make(map[rune]color.RGBA, len(glyphs))
	for _, g := range glyphs {
		table[g.Label] = g.Color
	}
	return table
}

// chunkType は1チャンクの場所の種別。全チャンクがいずれか1つに分類され、暗黙の既定を持たない。
// 特徴的な地物が当たらないチャンクは消極的な「残り」でなく、明示的に荒れ地に分類される。
// 実体は文字列。%v やログで数値でなく種別名が出て、デバッグで読みやすい。専用の String は要らない。
type chunkType string

const (
	chunkWasteland       chunkType = "wasteland"        // 荒れ地。特徴的な地物が無い開けた地形
	chunkSettlement      chunkType = "settlement"       // 集落。村・一軒家
	chunkUrban           chunkType = "urban"            // 市街地。建物チャンク
	chunkDungeonEntrance chunkType = "dungeon_entrance" // 遺跡入口
	chunkLandmark        chunkType = "landmark"         // 自然の点在ランドマーク
)

// chunkTypeAt は c の種別を返す純関数。全チャンクを漏れなく分類し、当たる地物が無ければ明示的に
// 荒れ地を返す。優先度は市街地 > 遺跡入口 > 集落 > 点在ランドマーク > 荒れ地。地図も生成もこの
// 分類を唯一の源にするので、地図の記号と実体が食い違わない。
func chunkTypeAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) chunkType {
	if _, ok := districtAt(raws, runSeed, c, cols); ok {
		return chunkUrban
	}
	if dungeonEntrancePlacement.At(runSeed, c, cols) {
		return chunkDungeonEntrance
	}
	if settlementPlacement.At(runSeed, c, cols) {
		return chunkSettlement
	}
	if landmarkPlacement.At(runSeed, c, cols) {
		return chunkLandmark
	}
	return chunkWasteland
}

// chunkPlace は1チャンクの種別を地図記号で返す純関数。
func chunkPlace(raws rawschema.Raws, cat zoneCatalog, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) rune {
	switch chunkTypeAt(raws, runSeed, c, cols) {
	case chunkUrban:
		r, _ := districtAt(raws, runSeed, c, cols)
		if g, ok := glyphByID(raws, urbanFacilityAt(cat, r, c)); ok {
			return g.Label
		}
		return placeUnknownGlyph
	case chunkDungeonEntrance:
		return chunkGlyph(raws, rawschema.DungeonEntrance)
	case chunkSettlement:
		if g, ok := glyphByID(raws, settlementAt(raws, runSeed, c).Id); ok {
			return g.Label
		}
		return placeUnknownGlyph
	case chunkLandmark:
		if g, ok := glyphByID(raws, landmarkKindAt(raws, runSeed, c)); ok {
			return g.Label
		}
		return placeUnknownGlyph
	case chunkWasteland:
		return chunkGlyph(raws, rawschema.Wasteland)
	}
	return placeUnknownGlyph
}

// chunkGlyph は持ち主の行を持たないチャンクの種類の記号を引く。未登録は保険の記号。
func chunkGlyph(raws rawschema.Raws, chunk rawschema.GlyphChunk) rune {
	if mg, ok := raw.GetChunkGlyph(raws, chunk); ok {
		return toGlyphInfo(mg).Label
	}
	return placeUnknownGlyph
}

// SchematicLegend は俯瞰図の文字と意味の対応表を返す。凡例をテストログや画面に添える。
func SchematicLegend(raws rawschema.Raws) string {
	glyphs := LegendGlyphs(raws)
	parts := make([]string, 0, len(glyphs))
	for _, g := range glyphs {
		parts = append(parts, fmt.Sprintf("%c %s", g.Label, g.Name))
	}
	return strings.Join(parts, "  ")
}
