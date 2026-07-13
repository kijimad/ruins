package route

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutgoing(t *testing.T) {
	t.Parallel()
	g := &Graph{
		Edges: []Edge{
			{From: 0, To: 1},
			{From: 0, To: 2},
			{From: 1, To: 3},
		},
	}
	assert.Len(t, g.Outgoing(0), 2)
	assert.Len(t, g.Outgoing(1), 1)
	assert.Empty(t, g.Outgoing(3))
}

func TestNodeByID(t *testing.T) {
	t.Parallel()
	g := &Graph{Nodes: []Node{{ID: 0, Type: NodeHome}, {ID: 1, Type: NodeRuin}}}
	assert.Equal(t, NodeRuin, g.NodeByID(1).Type)
	assert.Nil(t, g.NodeByID(99))
}

func TestResolveLeg_LoadIncreasesFoodCost(t *testing.T) {
	t.Parallel()
	edge := Edge{Type: EdgeNormal, Faces: 3}
	light := ResolveLeg(edge, 0)
	heavy := ResolveLeg(edge, 100)
	// 積載が重いほど1面の食料消費が増える（運搬役が積荷を食う＝物量で頂点が生まれる）
	assert.Greater(t, heavy.Cost.Food, light.Cost.Food)
}

func TestResolveLeg_FrontAdvanceEqualsFaces(t *testing.T) {
	t.Parallel()
	// 寒波前線は面数ぶん前進する（位置・前線は同一単位＝累積面数で扱う）
	r := ResolveLeg(Edge{Type: EdgeShortcut, Faces: 2}, 0)
	assert.Equal(t, 2, r.FrontAdvance)
}

func TestResolveLeg_ShortcutTradeoff(t *testing.T) {
	t.Parallel()
	shortcut := ResolveLeg(Edge{Type: EdgeShortcut, Faces: EdgeShortcut.baseFaces()}, 0)
	detour := ResolveLeg(Edge{Type: EdgeDetour, Faces: EdgeDetour.baseFaces()}, 0)
	// 近道は冷え、遭遇が濃い。迂回は暖かく安全寄り（逆向き利得）
	assert.Less(t, shortcut.TempDelta, detour.TempDelta)
	assert.Greater(t, shortcut.EncounterChance, detour.EncounterChance)
	// ただし近道は面数が少ない＝供給を食わず寒波接近も少ない
	assert.Less(t, shortcut.FrontAdvance, detour.FrontAdvance)
}
