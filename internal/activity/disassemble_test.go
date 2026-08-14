package activity

import (
	"math/rand/v2"
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mlange-42/ark/ecs"
)

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
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
		require.NoError(t, err)
		_, err = lifecycle.SpawnBackpackItem(world, "iron_cutter", 1)
		require.NoError(t, err)

		grade, name, ok := FindBestDisassemblyTool(world, player, oapi.Prying)
		require.True(t, ok)
		assert.Equal(t, 1, grade)
		assert.Equal(t, "Monkey Wrench", name)

		grade, name, ok = FindBestDisassemblyTool(world, player, oapi.Cutting)
		require.True(t, ok)
		assert.Equal(t, 3, grade)
		assert.Equal(t, "Iron Cutter", name)
	})

	t.Run("分類に適合する工具がなければ見つからない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
		require.NoError(t, err)

		_, _, ok := FindBestDisassemblyTool(world, player, oapi.Precision)
		assert.False(t, ok)
	})
}

func TestDisassembleBehavior_Validate_工具がないとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := &gc.Activity{Params: &gc.DisassembleParams{Target: crate}}
	err = da.Validate(comp, player, world)
	var ve *UserError
	require.ErrorAs(t, err, &ve)
}

func TestDisassembleBehavior_Validate_分解定義のない対象はエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	desk, err := lifecycle.SpawnProp(world, "desk", 11, 10)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(desk, player, world)
	err = da.Validate(comp, player, world)
	require.Error(t, err)
	assert.ErrorContains(t, err, "disassembly definition")
}

func TestDisassembleBehavior_propを分解すると素材が足元に落ちる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	assert.Contains(t, world.Components.Interactable.Get(crate).Interactions, gc.InteractionDisassemble,
		"分解定義を持つpropはInteractionDisassembleを持つべき")

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(crate, player, world)
	// baseAP2000 スキル0 グレード1 で必要AP2000
	assert.Equal(t, 2000, comp.Progress.Max)

	err = da.Validate(comp, player, world)
	require.NoError(t, err)
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
	assert.Contains(t, fieldNames, "Hardwood")

	// 分解完了で機械スキルの経験値が入る
	mechanic := world.Components.Skills.Get(player).Get(gc.SkillMechanic)
	assert.Positive(t, mechanic.Exp.Current)
}

func TestDisassembleBehavior_アイテムを分解すると消費して素材が所持品に入る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "electric_screwdriver", 1)
	require.NoError(t, err)
	hdd, err := lifecycle.SpawnBackpackItem(world, "hard_disk", 1)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(hdd, player, world)
	// baseAP1000 グレード2 で必要AP800
	assert.Equal(t, 800, comp.Progress.Max)

	require.NoError(t, da.Validate(comp, player, world))
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
	assert.Contains(t, backpackNames, "Scrap Iron", "確定枠の鉄くずが所持品に入るべき")
}

func TestDisassembleBehavior_収納propを分解すると中身が足元に出る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 11, 10)
	require.NoError(t, err)

	// 収納に中身を入れる
	loot, err := lifecycle.SpawnFieldItem(world, "bread", 12, 12, 1)
	require.NoError(t, err)
	require.NoError(t, lifecycle.MoveToStorage(world, loot, crate))

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(crate, player, world)
	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))
	for comp.State == gc.ActivityStateRunning {
		require.NoError(t, da.DoTurn(comp, player, world))
	}
	require.NoError(t, da.Finish(comp, player, world))

	assert.False(t, world.ECS.Alive(crate), "分解した木箱は消えるべき")
	require.True(t, world.ECS.Alive(loot), "中身は孤児化せず残るべき")
	assert.True(t, world.Components.LocationOnField.Has(loot), "中身はフィールドに出るべき")
	lootGrid := world.Components.GridElement.Get(loot)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 11, Y: 10}, lootGrid.Coord, "中身は木箱の足元に落ちるべき")
}

func TestDisassembleBehavior_Finish_対象が既に消えていれば何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(crate, player, world)
	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))

	world.ECS.RemoveEntity(crate)

	require.NoError(t, da.Finish(comp, player, world))

	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	count := 0
	for q.Next() {
		count++
	}
	assert.Equal(t, 0, count, "消えた対象から産出が湧かないべき")
}

func TestDisassembleBehavior_スタックのあるアイテムは1個だけ消費する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "electric_screwdriver", 1)
	require.NoError(t, err)
	hdd, err := lifecycle.SpawnBackpackItem(world, "hard_disk", 2)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(hdd, player, world)
	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))
	for comp.State == gc.ActivityStateRunning {
		require.NoError(t, da.DoTurn(comp, player, world))
	}
	require.NoError(t, da.Finish(comp, player, world))

	require.True(t, world.ECS.Alive(hdd), "残数があるうちはエンティティが消えないべき")
	assert.Equal(t, 1, query.GetEntityCount(world, hdd), "1個だけ消費されるべき")
}

func TestDisassembleBehavior_Finish_レベルアップでStatsChangedが付く(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// 次の獲得でレベルアップする直前まで経験値を積んでおく
	mechanic := world.Components.Skills.Get(player).Get(gc.SkillMechanic)
	mechanic.Exp.Current = mechanic.Exp.Max - 1

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(crate, player, world)
	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))
	for comp.State == gc.ActivityStateRunning {
		require.NoError(t, da.DoTurn(comp, player, world))
	}
	require.NoError(t, da.Finish(comp, player, world))

	assert.GreaterOrEqual(t, mechanic.Value, 1, "機械スキルがレベルアップするべき")
	assert.True(t, world.Components.StatsChanged.Has(player),
		"レベルアップ時はステータス再計算マーカーが付くべき")
}

func TestDisassembleBehavior_Validate_敵が隣接していると開始できない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 9, Y: 10}, "fireball")
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := &gc.Activity{Params: &gc.DisassembleParams{Target: crate}}
	err = da.Validate(comp, player, world)
	var ve *UserError
	require.ErrorAs(t, err, &ve)
}

func TestDisassembleBehavior_DoTurn_敵が接近すると中断する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(crate, player, world)
	err = da.Validate(comp, player, world)
	require.NoError(t, err)
	require.NoError(t, da.Start(comp, player, world))
	require.NoError(t, da.DoTurn(comp, player, world))
	require.Equal(t, gc.ActivityStateRunning, comp.State, "敵がいなければ継続するべき")

	// 分解の途中で敵が隣接タイルまで近づいてきた
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 11}, "fireball")
	require.NoError(t, err)

	require.NoError(t, da.DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, "disassembly interrupted because enemies are nearby", comp.CancelReason)
}

func TestDisassembleBehavior_Info(t *testing.T) {
	t.Parallel()

	db := &DisassembleBehavior{}
	info := db.Info()

	assert.Equal(t, "Disassemble", info.Name)
	assert.True(t, info.Interruptible)
	assert.True(t, info.Resumable)
}

func TestDisassembleBehavior_Name(t *testing.T) {
	t.Parallel()

	db := &DisassembleBehavior{}
	assert.Equal(t, gc.BehaviorDisassemble, db.Name())
}

func TestYieldsMarkup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stacks []lifecycle.YieldStack
		want   string
	}{
		{"1件は単独表記", []lifecycle.YieldStack{{Name: "鉄くず", Count: 2}}, "鉄くず x2"},
		{"複数件は読点で連結", []lifecycle.YieldStack{{Name: "鉄くず", Count: 2}, {Name: "硬木", Count: 1}}, "鉄くず x2, 硬木 x1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			// マークアップを描画した後の表示テキストが期待どおり並ぶことを確認する
			var got strings.Builder
			for _, f := range gamelog.ParseMarkup(yieldsMarkup(tt.stacks, world)) {
				got.WriteString(f.Text)
			}
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestDisassembleBehavior_DoTurn_対象が消えると中断する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(crate, player, world)
	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))

	world.ECS.RemoveEntity(crate)

	require.NoError(t, da.DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, "interrupted because the disassembly target disappeared", comp.CancelReason)

	// 対象が消えた後のキャンセル処理でもエラーにならない
	require.NoError(t, da.Canceled(comp, player, world))
}

func TestDisassembleBehavior_DoTurn_工具を失うと中断する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	wrench, err := lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleBehavior{}
	comp := NewDisassembleActivity(crate, player, world)
	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))

	world.ECS.RemoveEntity(wrench)

	require.NoError(t, da.DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, "disassembly interrupted because the tool was lost", comp.CancelReason)
}
