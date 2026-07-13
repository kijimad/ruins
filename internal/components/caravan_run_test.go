package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/route"
	"github.com/stretchr/testify/assert"
)

func TestNewCaravanRun(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(42, route.ExpeditionDeepVault)

	assert.NotNil(t, cr.Graph)
	assert.Equal(t, cr.Graph.Home, cr.Current, "起点は母港であること")
	assert.Equal(t, []route.NodeID{cr.Graph.Home}, cr.Visited, "初期の通過済みは母港のみ")
	assert.Equal(t, uint64(42), cr.Seed)
	assert.Equal(t, route.ExpeditionDeepVault, cr.Expedition)
}

func TestCaravanRun_GraphRebuildableFromSeed(t *testing.T) {
	t.Parallel()
	// Seed と Expedition から Graph を再構築でき、保存を省いた Graph を復元できる前提
	cr := NewCaravanRun(7, route.ExpeditionTradeCity)
	rebuilt := route.Generate(cr.Expedition, cr.Seed)
	assert.Equal(t, cr.Graph, rebuilt)
}

func TestCaravanRun_FrontLead(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	cr.CaravanProgress = 8
	cr.FrontProgress = 3
	assert.Equal(t, InitialFrontLead+5, cr.FrontLead(), "リード＝初期頭金＋前進−前線位置")
}

func TestCaravanRun_TravelKeepsLead(t *testing.T) {
	t.Parallel()
	// 移動は前線と等速なのでリードは変わらない（漏れバケツ回避）
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	before := cr.FrontLead()
	cr.AdvanceAlong(route.Edge{To: 99, Type: route.EdgeNormal, Faces: 3})
	assert.Equal(t, before, cr.FrontLead())
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

func TestCaravanRun_AdvanceAlong(t *testing.T) {
	t.Parallel()
	cr := NewCaravanRun(1, route.ExpeditionFrontier)
	cr.Supply = CaravanSupply{Food: 100, Fuel: 50, Cargo: 40}
	edge := route.Edge{From: cr.Current, To: 99, Type: route.EdgeNormal, Faces: 3}

	res := cr.AdvanceAlong(edge)

	assert.Equal(t, route.NodeID(99), cr.Current, "現在ノードが移動先へ更新される")
	assert.Equal(t, route.NodeID(99), cr.Visited[len(cr.Visited)-1], "移動先が通過済みに追加される")
	assert.Equal(t, 3, cr.CaravanProgress, "累積面数が面数ぶん前進する")
	assert.Equal(t, 3, cr.FrontProgress, "寒波前線も面数ぶん前進する")
	assert.Equal(t, 100-res.Cost.Food, cr.Supply.Food, "食料が消費ぶん減る")
	assert.Equal(t, 50-res.Cost.Fuel, cr.Supply.Fuel, "燃料が消費ぶん減る")
	assert.Greater(t, res.Cost.Food, 3*2, "積載40で1面消費が基本値2を上回る（頂点）")
}
