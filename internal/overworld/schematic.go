package overworld

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// チャンクマップ表記は、タイルの材質ではなく「そのマスがどの種別の場所か」を1文字で示す
// 俯瞰図。CDDA のオーバーマップが場所ごとに文字を割り当てる流儀の翻案で、市街地の建物を
// 施設の種別で塗り分ける。生成は (runSeed, 座標) の純関数なので、ECS のタイルを生成せずに
// 種別だけを算出できる。生成挙動の確認とゴールデンテストの可読な表現に使う。

// GlyphInfo は種別マップ上の1文字表記と、凡例に出す名前。UI の着色や凡例に使う。
type GlyphInfo struct {
	Label rune
	Name  string
}

// facilityGlyphs は施設種別の1文字表記。文字は種別の頭文字などから選び、衝突を避ける。
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

// 地物レベルの表記。建物より粗い単位で、チャンク中心などに置く。UI と共有するため公開する。
const (
	GlyphField   = '.' // 原野
	GlyphRoad    = '=' // 舗装路
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

// ChunkSchematic は1チャンクを種別文字で表す俯瞰図を返す。各行が1文字ずつ chunkW 分並び、
// chunkH 行になる。市街地の建物はその施設種別の文字で塗り、原野は '.' にする。
// 遺跡入口・集落・点在POIはチャンク中心に地物の文字を置く。純関数で ECS を生成しない。
func ChunkSchematic(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk, chunkW, chunkH consts.Tile) []string {
	grid := make([][]rune, chunkH)
	for y := range grid {
		grid[y] = make([]rune, chunkW)
		for x := range grid[y] {
			grid[y][x] = GlyphField
		}
	}
	set := func(x, y consts.Tile, r rune) {
		if x >= 0 && x < chunkW && y >= 0 && y < chunkH {
			grid[y][x] = r
		}
	}

	// 地物レベルの印はチャンク中心へ。建物より優先度が低いので先に置き、市街地で上書きする。
	// 配置判定の相互排他は生成側と同じなので、複数が同一チャンクで重なることはまれ
	cx, cy := chunkW/2, chunkH/2
	switch {
	case ruinPlacement.At(runSeed, c, rows):
		set(cx, cy, GlyphRuin)
	case settlementPlacement.At(runSeed, c, rows):
		// 開始特例は俯瞰図に無関係なので、開始チャンクを含まない規模抽選をそのまま使う
		if settlementVillageRoll(runSeed, c) {
			set(cx, cy, GlyphVillage)
		} else {
			set(cx, cy, GlyphHamlet)
		}
	case poiPlacement.At(runSeed, c, rows):
		set(cx, cy, GlyphPOI)
	}

	// 市街地の建物を施設種別の文字で塗る。断片に入る footprint だけをクリップする
	if anchor, width, ok := urbanAnchorOf(runSeed, c, rows); ok {
		citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
		buildings := cityLayout(citySeed, width, width.Tiles(chunkW), chunkH)
		fragOrigin := (c.X - anchor.X).Tiles(chunkW)
		for _, b := range buildings {
			g, ok := facilityGlyphs[facilityCatalog[b.facility].kind]
			label := GlyphUnknown
			if ok {
				label = g.Label
			}
			for ly := b.y; ly < b.y+b.h; ly++ {
				for lx := b.x; lx < b.x+b.w; lx++ {
					set(lx-fragOrigin, ly, label)
				}
			}
		}
	}

	out := make([]string, chunkH)
	for y := range grid {
		out[y] = string(grid[y])
	}
	return out
}

// RegionSchematic は連続する numChunks 個のチャンクを東へ並べた俯瞰図を返す。
// c0 が西端チャンク。市街地が複数チャンクにまたがる様子を1枚で確認できる。
func RegionSchematic(runSeed uint64, c0 worldstream.ChunkCoord, numChunks int, rows consts.Chunk, chunkW, chunkH consts.Tile) []string {
	rowsOut := make([]string, chunkH)
	for i := range numChunks {
		c := worldstream.ChunkCoord{X: c0.X + consts.Chunk(i), Y: c0.Y}
		chunk := ChunkSchematic(runSeed, c, rows, chunkW, chunkH)
		for y := range chunk {
			rowsOut[y] += chunk[y]
		}
	}
	return rowsOut
}

// SchematicLegend は俯瞰図の文字と意味の対応表を返す。凡例をテストログや画面に添える。
func SchematicLegend() string {
	var b strings.Builder
	// 舗装路 GlyphRoad はまだ俯瞰図に描かないので凡例からも外す。道の描画を入れたら戻す
	fmt.Fprintf(&b, "%c 原野  %c 村  %c 一軒家  %c 遺跡入口  %c 点在POI\n",
		GlyphField, GlyphVillage, GlyphHamlet, GlyphRuin, GlyphPOI)
	// 施設は種別の enum 順で安定させる
	kinds := make([]facilityKind, 0, len(facilityGlyphs))
	for k := range facilityGlyphs {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		g := facilityGlyphs[k]
		parts = append(parts, fmt.Sprintf("%c %s", g.Label, g.Name))
	}
	b.WriteString(strings.Join(parts, "  "))
	return b.String()
}
