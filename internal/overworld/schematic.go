package overworld

import (
	"fmt"
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

// 俯瞰図の記号は尺度の違う2層の語彙でできている。両者は対等な兄弟ではなく、市街地以外の粗い層と
// 市街地の中の細かい層という入れ子の関係にある。地図はチャンクの種別に応じてどちらかの記号を1つ選ぶ。
//   - placeType: チャンク尺度。市街地以外のチャンクを1記号で表す。荒れ地・村・一軒家・遺跡入口・
//     点在POI。地図の表示専用で、生成には関与しない。
//   - facilityType: 建物尺度。市街地チャンクの中の1建物の種別。住宅・商店・診療所など。表示だけでなく
//     市街地生成の重み抽選にも使う実体のあるドメイン型で、urban.go が持つ。
// 地図は市街地チャンクを facilityType の記号で、それ以外を placeType の記号で描く。凡例 LegendGlyphs は
// チャンク尺度に続けて建物尺度を並べ、2層を1つの表にする。

// facilityGlyphs は施設種別の1文字表記。1マスに1文字で建物の種別を示す。
// 武器屋にあたるのは現代日本設定では骨董品店で、A で表す。
var facilityGlyphs = map[facilityType]GlyphInfo{
	facilityHouse:   {'h', "住宅"},
	facilityStore:   {'S', "商店"},
	facilityOffice:  {'O', "事務所"},
	facilityDepot:   {'D', "倉庫"},
	facilityAntique: {'A', "骨董品店"},
	facilityClinic:  {'C', "診療所"},
	facilityLab:     {'L', "研究施設"},
}

// facilityOrder は凡例に出す施設種別を表示順で並べる。placeOrder と同じく、map は順序を持たない
// ので順序だけ別に定義する。string 化で facilityType は辞書順に落ちるため、表示順はここで固定する。
var facilityOrder = []facilityType{
	facilityHouse, facilityStore, facilityOffice, facilityDepot, facilityAntique, facilityClinic, facilityLab,
}

// placeType はチャンク尺度の記号キー。市街地以外のチャンクを1記号で表す表示専用の分類で、記号と
// 凡例名を placeGlyphs から引くために使う。生成には関与しない。chunkType とは1対1ではない。
// chunkSettlement は村ロールで placeVillage と placeHamlet に分かれ、chunkUrban は建物尺度の
// facilityType を使うのでここには無い。実体は文字列。%v やログで数値でなく種別名が出て読みやすい。
type placeType string

const (
	placeField   placeType = "field"   // 荒れ地
	placeVillage placeType = "village" // 村
	placeHamlet  placeType = "hamlet"  // 一軒家
	placeRuin    placeType = "ruin"    // 遺跡入口
	placePOI     placeType = "poi"     // 点在POI
	placeUnknown placeType = "unknown" // 分類漏れの保険。凡例には出さない
)

// placeGlyphs は地物種別の1文字表記と凡例名。facilityGlyphs と同じ形で、記号と名前を1箇所に
// 集約する。UI の着色や凡例はこれ1つを源にし、記号や名前を別の箇所へ直書きしない。
var placeGlyphs = map[placeType]GlyphInfo{
	placeField:   {'.', "荒れ地"},
	placeVillage: {'T', "村"},
	placeHamlet:  {'t', "一軒家"},
	placeRuin:    {'>', "遺跡入口"},
	placePOI:     {'*', "点在POI"},
	placeUnknown: {'?', "未分類"},
}

// placeOrder は凡例に出す地物種別を表示順で並べる。map は順序を持たないので順序だけ別に定義する。
// placeUnknown は分類漏れの保険なので凡例には含めない。
var placeOrder = []placeType{placeField, placeVillage, placeHamlet, placeRuin, placePOI}

// LegendGlyphs は俯瞰図の全記号と凡例名を表示順で返す。地物レベルに続けて施設レベルを並べる。
// SchematicLegend も UI の凡例もこれ1つを源にし、名前をあちこちに直書きしない。
func LegendGlyphs() []GlyphInfo {
	return append(PlaceGlyphs(), FacilityGlyphs()...)
}

// PlaceGlyphs は地物種別の記号と名前を表示順で返す。UI の凡例や着色で使う。FacilityGlyphs と対で、
// 施設側と同じ形で地物種別を扱えるようにする。分類漏れの保険 placeUnknown は含めない。
func PlaceGlyphs() []GlyphInfo {
	out := make([]GlyphInfo, 0, len(placeOrder))
	for _, k := range placeOrder {
		out = append(out, placeGlyphs[k])
	}
	return out
}

// FacilityGlyphs は施設種別の文字と名前を表示順で返す。UI の凡例や着色で建物を種別ごとに
// 扱うために使う。地物レベルの記号は PlaceGlyphs が対で返す。
func FacilityGlyphs() []GlyphInfo {
	out := make([]GlyphInfo, 0, len(facilityOrder))
	for _, k := range facilityOrder {
		out = append(out, facilityGlyphs[k])
	}
	return out
}

// chunkType は1チャンクの場所の種別。全チャンクがいずれか1つに分類され、暗黙の既定を持たない。
// 特徴的な地物が当たらないチャンクは消極的な「残り」でなく、明示的に荒れ地に分類される。
// 実体は文字列。%v やログで数値でなく種別名が出て、デバッグで読みやすい。専用の String は要らない。
type chunkType string

const (
	chunkWasteland    chunkType = "wasteland"     // 荒れ地。特徴的な地物が無い開けた地形
	chunkSettlement   chunkType = "settlement"    // 集落。村・一軒家
	chunkUrban        chunkType = "urban"         // 市街地。建物チャンク
	chunkRuinEntrance chunkType = "ruin_entrance" // 遺跡入口
	chunkPOI          chunkType = "poi"           // 自然の点在POI
)

// chunkTypeAt は c の種別を返す純関数。全チャンクを漏れなく分類し、当たる地物が無ければ明示的に
// 荒れ地を返す。優先度は市街地 > 遺跡入口 > 集落 > 点在POI > 荒れ地。地図も生成もこの分類を
// 唯一の源にするので、地図の記号と実体が食い違わない。
func chunkTypeAt(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk) chunkType {
	if _, _, ok := urbanChunkInfo(runSeed, c, rows); ok {
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
		kind, _, _ := urbanChunkInfo(runSeed, c, rows)
		if g, ok := facilityGlyphs[kind]; ok {
			return g.Label
		}
		return placeGlyphs[placeUnknown].Label
	case chunkRuinEntrance:
		return placeGlyphs[placeRuin].Label
	case chunkSettlement:
		if settlementVillageRoll(runSeed, c) {
			return placeGlyphs[placeVillage].Label
		}
		return placeGlyphs[placeHamlet].Label
	case chunkPOI:
		return placeGlyphs[placePOI].Label
	case chunkWasteland:
		return placeGlyphs[placeField].Label
	}
	return placeGlyphs[placeUnknown].Label
}

// SchematicLegend は俯瞰図の文字と意味の対応表を返す。凡例をテストログや画面に添える。
func SchematicLegend() string {
	glyphs := LegendGlyphs()
	parts := make([]string, 0, len(glyphs))
	for _, g := range glyphs {
		parts = append(parts, fmt.Sprintf("%c %s", g.Label, g.Name))
	}
	return strings.Join(parts, "  ")
}
