package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/route"
	"github.com/stretchr/testify/assert"
)

func TestNewCaravanRun(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(42, route.ExpeditionDeepVault)

	assert.NotNil(t, cr.Beacons)
	assert.Equal(t, cr.Beacons.Home, cr.Current, "起点は母港であること")
	assert.Equal(t, InitialFrontLead, cr.FrontLead(), "初期リード")
	assert.Equal(t, uint64(42), cr.Seed)
	assert.Equal(t, route.ExpeditionDeepVault, cr.Expedition)
}

func TestCaravanRun_BeaconsRebuildableFromSeed(t *testing.T) {
	t.Parallel()
	// Seed と Expedition から停留点マップを再構築でき、保存を省いた Beacons を復元できる前提
	cr := NewCaravanRun(7, route.ExpeditionTradeCity)
	rebuilt := route.GenerateBeacons(cr.Expedition, cr.Seed)
	assert.Equal(t, cr.Beacons, rebuilt)
}

func TestCaravanRun_ForwardJumpKeepsLead(t *testing.T) {
	t.Parallel()
	// ジャンプ（前進）は前線と等速なのでリードは変わらない（漏れバケツ回避）
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	before := cr.FrontLead()
	next := cr.Beacons.Outgoing(cr.Current)
	cr.JumpTo(next[0])
	assert.Equal(t, before, cr.FrontLead(), "前進はリード不変")
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
	// 食料が尽きた状態でジャンプすると、寒波前線が余分に詰めてリードを縮める（食料＝射程）
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	cr.Supply.Food = 0 // 飢餓状態
	before := cr.FrontLead()
	next := cr.Beacons.Outgoing(cr.Current)
	cr.JumpTo(next[0])
	// 通常の前進ならリード不変だが、飢餓ぶん余分に縮む
	assert.Equal(t, before-StarvationFrontPenalty, cr.FrontLead())
	assert.True(t, cr.IsStarving())
	assert.GreaterOrEqual(t, cr.Supply.Food, 0, "食料は0未満にならない")
}

func TestCaravanRun_JumpConsumesSupplyAndAdvances(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	cr.Supply = CaravanSupply{Food: 100, Fuel: 50, Cargo: 0}
	next := cr.Beacons.Outgoing(cr.Current)
	cr.JumpTo(next[0])
	assert.Equal(t, 100-jumpFoodCost, cr.Supply.Food, "食料が消費ぶん減る")
	assert.Equal(t, 50-jumpFuelCost, cr.Supply.Fuel, "燃料が消費ぶん減る")
	assert.Equal(t, next[0], cr.Current, "現在停留点が移動先へ更新される")
	assert.Equal(t, 1, cr.CaravanProgress, "前進が1列進む")
}
