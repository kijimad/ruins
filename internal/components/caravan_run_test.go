package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/route"
	"github.com/stretchr/testify/assert"
)

func TestNewCaravanRun(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(42, route.ExpeditionDeepVault)

	assert.NotNil(t, cr.Grid)
	assert.Equal(t, cr.Grid.Home, cr.Pos, "起点は母港であること")
	assert.Equal(t, cr.Grid.Home.X-InitialFrontLead, cr.FrontCol, "前線は初期リードぶん後方")
	assert.Equal(t, InitialFrontLead, cr.FrontLead(), "初期リード")
	assert.Equal(t, uint64(42), cr.Seed)
	assert.Equal(t, route.ExpeditionDeepVault, cr.Expedition)
}

func TestCaravanRun_GridRebuildableFromSeed(t *testing.T) {
	t.Parallel()
	// Seed と Expedition から Grid を再構築でき、保存を省いた Grid を復元できる前提
	cr := NewCaravanRun(7, route.ExpeditionTradeCity)
	rebuilt := route.GenerateGrid(cr.Expedition, cr.Seed, GridW, GridH)
	assert.Equal(t, cr.Grid, rebuilt)
}

func TestCaravanRun_ForwardMoveKeepsLead(t *testing.T) {
	t.Parallel()
	// 右（前進）への移動は前線と等速なのでリードは変わらない（漏れバケツ回避）
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	before := cr.FrontLead()
	cr.MoveTo(route.Coord{X: cr.Pos.X + 1, Y: cr.Pos.Y})
	assert.Equal(t, before, cr.FrontLead(), "前進はリード不変")
}

func TestCaravanRun_SidewaysMoveShrinksLead(t *testing.T) {
	t.Parallel()
	// 上下（横）への移動は前進しないので前線だけ詰まりリードが1縮む
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	before := cr.FrontLead()
	cr.MoveTo(route.Coord{X: cr.Pos.X, Y: cr.Pos.Y + 1})
	assert.Equal(t, before-1, cr.FrontLead(), "横移動はリードが縮む")
}

func TestCaravanRun_DawdleShrinksLead(t *testing.T) {
	t.Parallel()
	// 道草（潜行・野営）は前線だけ詰めてリードを縮める
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	before := cr.FrontLead()
	cr.Dawdle(5)
	assert.Equal(t, before-5, cr.FrontLead())
	assert.False(t, cr.Swallowed())
}

func TestCaravanRun_Swallowed(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	cr.Dawdle(InitialFrontLead) // リードを0まで詰める
	assert.True(t, cr.Swallowed(), "リード0以下で呑まれ")
}

func TestCaravanRun_StarvationAcceleratesFront(t *testing.T) {
	t.Parallel()
	// 食料が尽きた状態で前進すると、寒波前線が余分に詰めてリードを縮める（食料＝射程）
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	cr.Supply.Food = 0 // 飢餓状態
	before := cr.FrontLead()
	cr.MoveTo(route.Coord{X: cr.Pos.X + 1, Y: cr.Pos.Y})
	// 通常の前進ならリード不変だが、飢餓ぶん余分に縮む
	assert.Equal(t, before-StarvationFrontPenalty, cr.FrontLead())
	assert.True(t, cr.IsStarving())
	assert.GreaterOrEqual(t, cr.Supply.Food, 0, "食料は0未満にならない")
}

func TestCaravanRun_MoveConsumesSupply(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	cr.Supply = CaravanSupply{Food: 100, Fuel: 50, Cargo: 0}
	cr.MoveTo(route.Coord{X: cr.Pos.X + 1, Y: cr.Pos.Y})
	assert.Equal(t, 100-moveFoodCost, cr.Supply.Food, "食料が消費ぶん減る")
	assert.Equal(t, 50-moveFuelCost, cr.Supply.Fuel, "燃料が消費ぶん減る")
	assert.Equal(t, route.Coord{X: 1, Y: cr.Grid.Home.Y}, cr.Pos, "現在セルが移動先へ更新される")
}

func TestCaravanRun_CanMoveTo(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	// 隣接・グリッド内・前線より前は可
	assert.True(t, cr.CanMoveTo(route.Coord{X: cr.Pos.X + 1, Y: cr.Pos.Y}), "右隣は可")
	assert.True(t, cr.CanMoveTo(route.Coord{X: cr.Pos.X, Y: cr.Pos.Y + 1}), "下隣は可")
	// 非隣接（斜め・2マス）は不可
	assert.False(t, cr.CanMoveTo(route.Coord{X: cr.Pos.X + 1, Y: cr.Pos.Y + 1}), "斜めは不可")
	assert.False(t, cr.CanMoveTo(route.Coord{X: cr.Pos.X + 2, Y: cr.Pos.Y}), "2マス先は不可")
	// 枠外は不可
	assert.False(t, cr.CanMoveTo(route.Coord{X: cr.Pos.X, Y: -1}), "枠外は不可")
	// 前線に追いつかれた列（凍結）へは不可
	cr.FrontCol = cr.Pos.X // 現在列まで前線を寄せる
	assert.False(t, cr.CanMoveTo(route.Coord{X: cr.Pos.X - 1, Y: cr.Pos.Y}), "凍結した後方へは不可")
}
