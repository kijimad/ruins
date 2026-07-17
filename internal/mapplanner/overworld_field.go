package mapplanner

import "github.com/kijimaD/ruins/internal/consts"

// OverworldFieldPlanner はオーバーワールドの「開けた地形」チャンクの初期プランナー。
//
// 部屋を掘る従来のダンジョンプランナーと逆で、全面通行可能をデフォルトにする（障壁は
// OverworldBarriers が例外的に置く）。これによりチャンクを東西に継いでも境界が壁で詰まらない。
// 詳細設計は docs/design/20260717_60.md §5.1。
type OverworldFieldPlanner struct{}

// PlanInitial は初期化を行う。開けた地形は部屋を持たないため何もしない。
func (OverworldFieldPlanner) PlanInitial(_ *MetaPlan) error { return nil }

// OverworldBarriers は開けた地形にまばらな障壁（稜線・岩）を置く MetaMapPlanner。
//
// design.md「到達不可地形が分岐・合流を作る」の実体。各ブロブの高さをマップ高さ未満に制限し、
// さらに「どの列も高さ全体を塞がない」ことを最後に保証して、東西の通行を構造的に守る。
type OverworldBarriers struct {
	// Density は障壁ブロブ数のタイル面積あたりの割合。0 なら既定値を使う
	Density float64
}

// PlanMeta は障壁を配置し、東西通行を保証する。
func (b OverworldBarriers) PlanMeta(planData *MetaPlan) error {
	w := int(planData.Level.TileWidth)
	h := int(planData.Level.TileHeight)
	if w <= 0 || h <= 0 {
		return nil
	}

	density := b.Density
	if density <= 0 {
		density = 0.004 // 面積の約0.4%を障壁ブロブの中心にする
	}
	wallTile := planData.GetTile(consts.TileNameWall)

	// まばらに障壁ブロブを置く。各ブロブの高さは h/3 までに制限し、単独では列を塞がない
	blobCount := int(float64(w*h) * density)
	for range blobCount {
		cx := planData.RNG.IntN(w)
		cy := planData.RNG.IntN(h)
		bw := 1 + planData.RNG.IntN(3)
		bh := 1 + planData.RNG.IntN(max(1, h/3))
		for dy := range bh {
			for dx := range bw {
				x, y := cx+dx, cy+dy
				if x < 0 || x >= w || y < 0 || y >= h {
					continue
				}
				planData.Tiles[planData.Level.XYTileIndex(consts.Tile(x), consts.Tile(y))] = wallTile
			}
		}
	}

	// 通行保証: 東西を貫く連結した通路を必ず1本掘る（連結性の構造保証）。
	// 「各列に通行可能タイルがある」だけでは E-W 経路を保証できない（列ごとに通行可能な y が
	// バラけて縦に途切れると4連結しない）。西端から東端まで連続した通路を constructive に
	// 掘ることで、障壁をどう置いても西端→東端が4連結することを保証する。
	carveEastWestPath(planData, w, h)
	return nil
}

// carveEastWestPath は西端から東端まで4連結した通路を1本掘る。
// 各列で通路タイルを通行可能にし、上下にずれる際は移動元・移動先の両タイルを通して縦の連結も確保する。
func carveEastWestPath(planData *MetaPlan, w, h int) {
	dirtTile := planData.GetTile("dirt")
	setDirt := func(x, y int) {
		if y < 0 || y >= h {
			return
		}
		planData.Tiles[planData.Level.XYTileIndex(consts.Tile(x), consts.Tile(y))] = dirtTile
	}

	y := h / 2
	for x := range w {
		setDirt(x, y)
		// 20% で上下に1歩蛇行する。ずれる際は移動元(既にdirt)と移動先の両方を通し縦連結を保つ
		if planData.RNG.IntN(5) == 0 {
			ny := y + (planData.RNG.IntN(2)*2 - 1) // y-1 または y+1
			if ny >= 0 && ny < h {
				setDirt(x, ny)
				y = ny
			}
		}
	}
}

// NewOverworldFieldPlanner は開けた地形（通行可能デフォルト＋まばらな障壁）のチェーンを作る。
func NewOverworldFieldPlanner(width, height consts.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(OverworldFieldPlanner{})
	chain.With(NewFillAll("dirt"))  // 全面を通行可能な地面で埋める（デフォルト通行可）
	chain.With(OverworldBarriers{}) // 障壁をまばらに置く（列は塞がない）
	return chain, nil
}
