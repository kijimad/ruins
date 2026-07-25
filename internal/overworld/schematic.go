package overworld

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// チャンクマップ表記は、1チャンク=1建物という縮尺に合わせ、各チャンクを「そこが何の種別の
// 場所か」の1文字で表す俯瞰図。CDDA のオーバーマップが OMT ごとに1記号を割り当てるのの
// 翻案。生成は (runSeed, 座標) の純関数なので、ECS のタイルを生成せず種別だけを算出できる。

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
	GlyphField   = '.' // 原野
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

// ChunkPlace は1チャンクの種別を1文字で返す純関数。市街地の建物チャンクはその施設種別、
// それ以外は当選した地物の記号、何もなければ原野を返す。地図と生成が同じ純関数から導かれる
// ので、地図の記号と実体が食い違わない。優先度は市街地 > 遺跡入口 > 集落 > 点在POI。
func ChunkPlace(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk) rune {
	if kind, _, ok := cityChunkInfo(runSeed, c, rows); ok {
		if g, ok := facilityGlyphs[kind]; ok {
			return g.Label
		}
		return GlyphUnknown
	}
	if ruinPlacement.At(runSeed, c, rows) {
		return GlyphRuin
	}
	if settlementPlacement.At(runSeed, c, rows) {
		if settlementVillageRoll(runSeed, c) {
			return GlyphVillage
		}
		return GlyphHamlet
	}
	if poiPlacement.At(runSeed, c, rows) {
		return GlyphPOI
	}
	return GlyphField
}

// SchematicLegend は俯瞰図の文字と意味の対応表を返す。凡例をテストログや画面に添える。
func SchematicLegend() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%c 原野  %c 村  %c 一軒家  %c 遺跡入口  %c 点在POI\n",
		GlyphField, GlyphVillage, GlyphHamlet, GlyphRuin, GlyphPOI)
	parts := make([]string, 0, len(facilityGlyphs))
	for _, g := range FacilityGlyphs() {
		parts = append(parts, fmt.Sprintf("%c %s", g.Label, g.Name))
	}
	b.WriteString(strings.Join(parts, "  "))
	return b.String()
}
