package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// newDisassembleTestPlayer は分解テスト用のプレイヤーを作る
func newDisassembleTestPlayer(world w.World) ecs.Entity {
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.TurnBased.Add(player, &gc.TurnBased{AP: gc.IntPool{Max: 100, Current: 100}})
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
	world.Components.Skills.Add(player, gc.NewSkills())
	return player
}

func TestRequiredDisassemblyAP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		baseAP    int
		skill     int
		toolGrade int
		want      int
	}{
		{"スキル0グレード1は等倍", 300, 0, 1, 300},
		{"スキル20で20%短縮しグレード2で80%になる", 300, 20, 2, 192},
		{"スキル短縮は50%で頭打ちになる", 300, 60, 3, 90},
		{"グレード2は80%になる", 300, 0, 2, 240},
		{"端数は切り上げる", 111, 1, 1, 110},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, RequiredDisassemblyAP(tt.baseAP, tt.skill, tt.toolGrade))
		})
	}
}

func TestFindBestDisassemblyTool(t *testing.T) {
	t.Parallel()

	t.Run("分類に適合する最高グレードの工具を選ぶ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := newDisassembleTestPlayer(world)

		_, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
		require.NoError(t, err)
		_, err = lifecycle.SpawnBackpackItem(world, "鉄カッター", 1)
		require.NoError(t, err)

		grade, name, ok := FindBestDisassemblyTool(world, player, oapi.Prying)
		require.True(t, ok)
		assert.Equal(t, 1, grade)
		assert.Equal(t, "モンキーレンチ", name)

		grade, name, ok = FindBestDisassemblyTool(world, player, oapi.Cutting)
		require.True(t, ok)
		assert.Equal(t, 3, grade)
		assert.Equal(t, "鉄カッター", name)
	})

	t.Run("分類に適合する工具がなければ見つからない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := newDisassembleTestPlayer(world)

		_, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
		require.NoError(t, err)

		_, _, ok := FindBestDisassemblyTool(world, player, oapi.Precision)
		assert.False(t, ok)
	})
}

func TestDisassembleActivity_Validate_工具がないとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player := newDisassembleTestPlayer(world)

	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleActivity{Target: crate}
	comp := &gc.Activity{Target: &crate}
	err = da.Validate(comp, player, world)
	require.Error(t, err)
	assert.ErrorContains(t, err, "工具")
}

func TestDisassembleActivity_BuildActivity_分解定義のない対象はエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player := newDisassembleTestPlayer(world)

	desk, err := lifecycle.SpawnProp(world, "desk", 11, 10)
	require.NoError(t, err)

	da := &DisassembleActivity{Target: desk}
	_, err = da.BuildActivity(player, world)
	require.Error(t, err)
	assert.ErrorContains(t, err, "分解定義")
}

func TestDisassembleActivity_propを分解すると素材が足元に落ちる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player := newDisassembleTestPlayer(world)

	_, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	assert.Contains(t, world.Components.Interactable.Get(crate).Interactions, gc.InteractionDisassemble,
		"分解定義を持つpropはInteractionDisassembleを持つべき")

	da := &DisassembleActivity{Target: crate}
	comp, err := da.BuildActivity(player, world)
	require.NoError(t, err)
	// baseAP200 スキル0 グレード1 で必要AP200、AP100につき2ターン
	assert.Equal(t, 2, comp.TurnsTotal)

	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))
	for comp.State == gc.ActivityStateRunning {
		require.NoError(t, da.DoTurn(comp, player, world))
	}
	require.Equal(t, gc.ActivityStateCompleted, comp.State)
	require.NoError(t, da.Finish(comp, player, world))

	assert.False(t, world.ECS.Alive(crate), "分解したpropは消えるべき")

	// 確定枠の硬木がフィールドに落ちる
	var fieldNames []string
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for q.Next() {
		fieldNames = append(fieldNames, query.GetEntityName(q.Entity(), world))
	}
	assert.Contains(t, fieldNames, "硬木")

	// 分解完了で機械スキルの経験値が入る
	mechanic := world.Components.Skills.Get(player).Get(gc.SkillMechanic)
	assert.Positive(t, mechanic.Exp.Current)
}

func TestDisassembleActivity_アイテムを分解すると消費して素材が所持品に入る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player := newDisassembleTestPlayer(world)

	_, err := lifecycle.SpawnBackpackItem(world, "電動ドライバー", 1)
	require.NoError(t, err)
	hdd, err := lifecycle.SpawnBackpackItem(world, "ハードディスク", 1)
	require.NoError(t, err)

	da := &DisassembleActivity{Target: hdd}
	comp, err := da.BuildActivity(player, world)
	require.NoError(t, err)
	// baseAP100 グレード2 で必要AP80、1ターンで終わる
	assert.Equal(t, 1, comp.TurnsTotal)

	require.NoError(t, da.Start(comp, player, world))
	for comp.State == gc.ActivityStateRunning {
		require.NoError(t, da.DoTurn(comp, player, world))
	}
	require.NoError(t, da.Finish(comp, player, world))

	assert.False(t, world.ECS.Alive(hdd), "最後の1個を分解したらエンティティが消えるべき")

	var backpackNames []string
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		backpackNames = append(backpackNames, query.GetEntityName(q.Entity(), world))
	}
	assert.Contains(t, backpackNames, "鉄くず", "確定枠の鉄くずが所持品に入るべき")
}

func TestDisassembleActivity_DoTurn_対象が消えると中断する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player := newDisassembleTestPlayer(world)

	_, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleActivity{Target: crate}
	comp, err := da.BuildActivity(player, world)
	require.NoError(t, err)
	require.NoError(t, da.Start(comp, player, world))

	world.ECS.RemoveEntity(crate)

	require.NoError(t, da.DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, "分解対象が消えたため中断", comp.CancelReason)
}

func TestDisassembleActivity_DoTurn_工具を失うと中断する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player := newDisassembleTestPlayer(world)

	wrench, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleActivity{Target: crate}
	comp, err := da.BuildActivity(player, world)
	require.NoError(t, err)
	require.NoError(t, da.Start(comp, player, world))

	world.ECS.RemoveEntity(wrench)

	require.NoError(t, da.DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, "工具を失ったため分解を中断", comp.CancelReason)
}
