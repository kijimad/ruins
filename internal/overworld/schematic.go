package overworld

import (
	"cmp"
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
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
//   - placeType: チャンク尺度。市街地以外のチャンクを1記号で表す。荒れ地・村・一軒家・遺跡入口・
//     点在ランドマークの各種別。地図の表示専用で、生成には関与しない。
//   - facilityType: 建物尺度。市街地チャンクの中の1建物の種別。住宅・商店・診療所など。表示だけでなく
//     市街地生成の重み抽選にも使う実体のあるドメイン型で、urban.go が持つ。
// 地図は市街地チャンクを facilityType の記号で、それ以外を placeType の記号で描く。凡例 LegendGlyphs は
// チャンク尺度に続けて建物尺度を並べ、2層を1つの表にする。

// placeType はチャンク尺度の記号キー。市街地以外のチャンクを1記号で表す表示専用の分類で、記号と
// 凡例名を mapGlyphs 行から引くために使う。生成には関与しない。chunkType とは1対1ではない。
// chunkSettlement は村ロールで placeVillage と placeHamlet に、chunkLandmark は種別ロールで
// 廃屋・農家跡・祠・キャンプ跡に分かれる。chunkUrban は建物尺度の施設 id を使うのでここには
// 無い。値は mapGlyphs の id と一致する。
type placeType string

const (
	placeField           placeType = "field"            // 荒れ地
	placeVillage         placeType = "village"          // 村
	placeHamlet          placeType = "hamlet"           // 一軒家
	placeDungeonEntrance placeType = "dungeon_entrance" // 遺跡入口
	placeAbandonedHut    placeType = "abandoned_hut"    // 点在ランドマーク: 廃屋
	placeFarmstead       placeType = "farmstead"        // 点在ランドマーク: 農家跡
	placeShrine          placeType = "shrine"           // 点在ランドマーク: 祠
	placeCampsite        placeType = "campsite"         // 点在ランドマーク: キャンプ跡
)

// placeUnknownGlyph は分類漏れの保険の記号。mapGlyphs には入れず凡例外なので Go に持つ。
const placeUnknownGlyph rune = '?'

// toGlyphInfo は mapGlyphs 行を GlyphInfo へ変換する。glyph は1文字の string なので先頭 rune を Label にする。
func toGlyphInfo(mg oapi.MapGlyph) GlyphInfo {
	label := placeUnknownGlyph
	if r := []rune(mg.Glyph); len(r) > 0 {
		label = r[0]
	}
	return GlyphInfo{Label: label, Name: mg.Name, Color: color.RGBA{R: mg.Color.R, G: mg.Color.G, B: mg.Color.B, A: mg.Color.A}}
}

// glyphByID は地物・施設の種別 id から地図記号を引く。mapGlyphs 行が単一出典。未登録は ok=false。
func glyphByID(raws oapi.Raws, id string) (GlyphInfo, bool) {
	mg, ok := raw.GetMapGlyph(raws, id)
	if !ok {
		return GlyphInfo{}, false
	}
	return toGlyphInfo(mg), true
}

// LegendGlyphs は俯瞰図の全記号と凡例名を order 順に返す。mapGlyphs 行が単一出典で、UI の凡例も
// SchematicLegend もこれを源にし、記号や名前を別の箇所へ直書きしない。
func LegendGlyphs(raws oapi.Raws) []GlyphInfo {
	mgs := raw.PtrSlice(raws.MapGlyphs)
	sorted := make([]oapi.MapGlyph, len(mgs))
	copy(sorted, mgs)
	slices.SortStableFunc(sorted, func(a, b oapi.MapGlyph) int { return cmp.Compare(a.Order, b.Order) })
	out := make([]GlyphInfo, 0, len(sorted))
	for _, mg := range sorted {
		out = append(out, toGlyphInfo(mg))
	}
	return out
}

// GlyphColor は種別文字に対応する色と、対応があるかを返す。凡例に出ない記号は ok=false になり、
// 未知記号の既定色は UI 側が決める。overworld は theme に依存しないので既定色を持たない。
func GlyphColor(raws oapi.Raws, r rune) (color.RGBA, bool) {
	for _, g := range LegendGlyphs(raws) {
		if g.Label == r {
			return g.Color, true
		}
	}
	return color.RGBA{}, false
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
func chunkTypeAt(runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) chunkType {
	if _, ok := urbanChunkAt(runSeed, c, cols); ok {
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

// ChunkPlace は1チャンクの種別を1文字で返す純関数。chunkTypeAt の分類を記号へ写す。市街地は
// 施設種別の記号、荒れ地は '.' を返す。種別を1つ足すと switch の網羅を linter が強制する。
func ChunkPlace(raws oapi.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) rune {
	switch chunkTypeAt(runSeed, c, cols) {
	case chunkUrban:
		// 施設種は urbanFacilityAt が raws から抽選する動的な値で、mapGlyphs に無い種が来うるので
		// ok チェックする。他の種別は placeType が局所で保証されるので直接引く
		kind, _ := urbanFacilityAt(raws, runSeed, c, cols)
		if g, ok := glyphByID(raws, kind); ok {
			return g.Label
		}
		return placeUnknownGlyph
	case chunkDungeonEntrance:
		return placeGlyph(raws, placeDungeonEntrance)
	case chunkSettlement:
		if settlementVillageRoll(runSeed, c) {
			return placeGlyph(raws, placeVillage)
		}
		return placeGlyph(raws, placeHamlet)
	case chunkLandmark:
		return placeGlyph(raws, landmarkPlaceType(landmarkKindAt(raws, runSeed, c)))
	case chunkWasteland:
		return placeGlyph(raws, placeField)
	}
	return placeUnknownGlyph
}

// placeGlyph は placeType の記号を mapGlyphs から引く。未登録は保険の記号へ落とす。
func placeGlyph(raws oapi.Raws, pt placeType) rune {
	if g, ok := glyphByID(raws, string(pt)); ok {
		return g.Label
	}
	return placeUnknownGlyph
}

// SchematicLegend は俯瞰図の文字と意味の対応表を返す。凡例をテストログや画面に添える。
func SchematicLegend(raws oapi.Raws) string {
	glyphs := LegendGlyphs(raws)
	parts := make([]string, 0, len(glyphs))
	for _, g := range glyphs {
		parts = append(parts, fmt.Sprintf("%c %s", g.Label, g.Name))
	}
	return strings.Join(parts, "  ")
}
