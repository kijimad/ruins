package overworld

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kijimaD/ruins/internal/consts"
)

// チャンクマップ表記は、1チャンク=1建物という縮尺に合わせ、各チャンクを「そこが何の種別の
// 場所か」の1文字で表す俯瞰図。1チャンク=1記号で場所の種別を示す。生成は (runSeed, 座標) の
// 純関数なので、ECS のタイルを生成せず種別だけを算出できる。

// GlyphInfo は種別マップ上の1文字表記と、凡例に出す名前。UI の着色や凡例に使う。
type GlyphInfo struct {
	Label rune
	Name  string
}

// facilityGlyphs は施設種別の1文字表記。1マスに1文字で建物の種別を示す。
// 武器屋にあたるのは現代日本設定では骨董品店で、A で表す。
var facilityGlyphs = map[facilityKind]GlyphInfo{
	facilityHouse:   {'h', "住宅"},
	facilityStore:   {'S', "商店"},
	facilityOffice:  {'O', "事務所"},
	facilityDepot:   {'D', "倉庫"},
	facilityAntique: {'A', "骨董品店"},
	facilityClinic:  {'C', "診療所"},
	facilityLab:     {'L', "研究施設"},
}

// 地物レベルの表記。1チャンクを1文字で表す。UI と共有するため公開する。
const (
	GlyphField   = '.' // 荒れ地
	GlyphVillage = 'T' // 村
	GlyphHamlet  = 't' // 一軒家
	GlyphRuin    = '>' // 遺跡入口
	GlyphPOI     = '*' // 自然の点在POI
	GlyphUnknown = '?' // 未分類
)

// FacilityGlyphs は施設種別の文字と名前を種別順で返す。UI の凡例や着色で建物を種別ごとに
// 扱うために使う。地物レベルの記号は Glyph 定数を直接参照する。
func FacilityGlyphs() []GlyphInfo {
	kinds := make([]facilityKind, 0, len(facilityGlyphs))
	for k := range facilityGlyphs {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	out := make([]GlyphInfo, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, facilityGlyphs[k])
	}
	return out
}

// chunkType は1チャンクの場所の種別。全チャンクがいずれか1つに分類され、暗黙の既定を持たない。
// 特徴的な地物が当たらないチャンクは消極的な「残り」でなく、明示的に荒れ地に分類される。
type chunkType uint8

const (
	chunkWasteland    chunkType = iota // 荒れ地。特徴的な地物が無い開けた地形
	chunkSettlement                    // 集落。村・一軒家
	chunkUrban                         // 市街地。建物チャンク
	chunkRuinEntrance                  // 遺跡入口
	chunkPOI                           // 自然の点在POI
)

// chunkTypeAt は c の種別を返す純関数。全チャンクを漏れなく分類し、当たる地物が無ければ明示的に
// 荒れ地を返す。優先度は市街地 > 遺跡入口 > 集落 > 点在POI > 荒れ地。地図も生成もこの分類を
// 唯一の源にするので、地図の記号と実体が食い違わない。
func chunkTypeAt(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk) chunkType {
	if _, _, ok := cityChunkInfo(runSeed, c, rows); ok {
		return chunkUrban
	}
	if ruinPlacement.At(runSeed, c, rows) {
		return chunkRuinEntrance
	}
	if settlementPlacement.At(runSeed, c, rows) {
		return chunkSettlement
	}
	if poiPlacement.At(runSeed, c, rows) {
		return chunkPOI
	}
	return chunkWasteland
}

// ChunkPlace は1チャンクの種別を1文字で返す純関数。chunkTypeAt の分類を記号へ写す。市街地は
// 施設種別の記号、荒れ地は '.' を返す。種別を1つ足すと switch の網羅を linter が強制する。
func ChunkPlace(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk) rune {
	switch chunkTypeAt(runSeed, c, rows) {
	case chunkUrban:
		kind, _, _ := cityChunkInfo(runSeed, c, rows)
		if g, ok := facilityGlyphs[kind]; ok {
			return g.Label
		}
		return GlyphUnknown
	case chunkRuinEntrance:
		return GlyphRuin
	case chunkSettlement:
		if settlementVillageRoll(runSeed, c) {
			return GlyphVillage
		}
		return GlyphHamlet
	case chunkPOI:
		return GlyphPOI
	case chunkWasteland:
		return GlyphField
	}
	return GlyphUnknown
}

// SchematicLegend は俯瞰図の文字と意味の対応表を返す。凡例をテストログや画面に添える。
func SchematicLegend() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%c 荒れ地  %c 村  %c 一軒家  %c 遺跡入口  %c 点在POI\n",
		GlyphField, GlyphVillage, GlyphHamlet, GlyphRuin, GlyphPOI)
	parts := make([]string, 0, len(facilityGlyphs))
	for _, g := range FacilityGlyphs() {
		parts = append(parts, fmt.Sprintf("%c %s", g.Label, g.Name))
	}
	b.WriteString(strings.Join(parts, "  "))
	return b.String()
}
