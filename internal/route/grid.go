package route

import "math/rand/v2"

// Coord はグリッド上のセル座標（左上原点）。
type Coord struct {
	X, Y int
}

// Grid は広域マップ（CDDA 風オーバーマップ）。各セルに地形（NodeType）を持つ 2D グリッド。
// キャラバンはセル単位で移動し、入ったセルの地形をエンゲージする。母港→目標を左右に置き、
// 寒波前線は左から列単位で迫る（背後セルは凍結＝戻れない＝一方向）。
//
// Graph（層状 DAG）に代わるマクロ移動のモデル。毎歩が方向選択になり選択肢が濃い。
type Grid struct {
	W, H  int
	Cells []NodeType // 行優先（len == W*H）
	Home  Coord      // 母港（出発）。左端中央
	Goal  Coord      // 目標。右端中央
}

// In は座標がグリッド内かを返す。
func (g *Grid) In(c Coord) bool {
	return c.X >= 0 && c.X < g.W && c.Y >= 0 && c.Y < g.H
}

// At は座標の地形を返す（範囲外は呼ばないこと）。
func (g *Grid) At(c Coord) NodeType {
	return g.Cells[c.Y*g.W+c.X]
}

// set は座標へ地形を書き込む（生成時のみ使用）。
func (g *Grid) set(c Coord, t NodeType) {
	g.Cells[c.Y*g.W+c.X] = t
}

// GenerateGrid は遠征とシードから W×H の地形グリッドを生成する純関数。
// 地形はフィールド優位（平原/山脈がベース・遺跡はたまに・街は稀）で、母港を左端中央・
// 目標を右端中央に置く。全セル通行可能なので母港→目標は常に到達可能。
func GenerateGrid(expedition ExpeditionType, seed uint64, w, h int) *Grid {
	rng := rand.New(rand.NewPCG(seed, 0))
	cells := make([]NodeType, w*h)
	for i := range cells {
		cells[i] = gridTerrain(expedition, rng)
	}
	g := &Grid{W: w, H: h, Cells: cells}
	g.Home = Coord{X: 0, Y: h / 2}
	g.Goal = Coord{X: w - 1, Y: h / 2}
	g.set(g.Home, NodeHome)
	g.set(g.Goal, NodeGoal)
	// 目標手前の列に前哨（最終補給/売却点）を1つ置く
	g.set(Coord{X: w - 2, Y: h / 2}, NodeOutpost)
	return g
}

// gridTerrain は広域マップ1セルの地形を選ぶ。大半は原野（平原/山脈）で、POI（遺跡/村/専門店）は
// 疎に散らす（CDDA のオーバーマップ＝ほとんど原野、ランドマークが点在）。遠征で POI 比率を調整する。
func gridTerrain(exp ExpeditionType, rng *rand.Rand) NodeType {
	ruinPct, marketPct, shopPct := 6, 4, 2
	switch exp {
	case ExpeditionDeepVault:
		ruinPct = 10 // 潜行重心（それでも大半は原野）
	case ExpeditionTradeCity:
		marketPct, shopPct = 8, 4 // 交易重心
	case ExpeditionPatron, ExpeditionFrontier:
		// 既定のまま
	}
	roll := rng.IntN(100)
	switch {
	case roll < ruinPct:
		return NodeRuin
	case roll < ruinPct+marketPct:
		return NodeMarket
	case roll < ruinPct+marketPct+shopPct:
		return NodeShop
	default:
		// 残りは原野（平原/山脈を半々）
		if rng.IntN(2) == 0 {
			return NodePlain
		}
		return NodeMountain
	}
}
