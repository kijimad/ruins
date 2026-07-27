package activity

import (
	"math/rand/v2"
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
	world.Components.FactionAlly.Add(player, &gc.FactionAlly{})
	return player
}

// spawnHostileAt は敵対エンティティを指定タイルに置く
func spawnHostileAt(world w.World, x consts.Tile, y consts.Tile) {
	hostile := world.ECS.NewEntity()
	world.Components.FactionEnemy.Add(hostile, &gc.FactionEnemy{})
	world.Components.GridElement.Add(hostile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
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
	// baseAP2000 スキル0 グレード1 で必要AP2000、AP100につき20ターン
	assert.Equal(t, 20, comp.TurnsTotal)

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
	// baseAP1000 グレード2 で必要AP800、AP100につき8ターン
	assert.Equal(t, 8, comp.TurnsTotal)

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

func TestDisassembleActivity_収納propを分解すると中身が足元に出る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player := newDisassembleTestPlayer(world)

	_, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "木箱", 11, 10)
	require.NoError(t, err)

	// 収納に中身を入れる
	loot, err := lifecycle.SpawnFieldItem(world, "パン", 12, 12, 1)
	require.NoError(t, err)
	require.NoError(t, lifecycle.MoveToStorage(world, loot, crate))

	da := &DisassembleActivity{Target: crate}
	comp, err := da.BuildActivity(player, world)
	require.NoError(t, err)
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

func TestDisassembleActivity_Finish_対象が既に消えていれば何もしない(t *testing.T) {
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

	require.NoError(t, da.Finish(comp, player, world))

	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	count := 0
	for q.Next() {
		count++
	}
	assert.Equal(t, 0, count, "消えた対象から産出が湧かないべき")
}

func TestDisassembleActivity_スタックのあるアイテムは1個だけ消費する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player := newDisassembleTestPlayer(world)

	_, err := lifecycle.SpawnBackpackItem(world, "電動ドライバー", 1)
	require.NoError(t, err)
	hdd, err := lifecycle.SpawnBackpackItem(world, "ハードディスク", 2)
	require.NoError(t, err)

	da := &DisassembleActivity{Target: hdd}
	comp, err := da.BuildActivity(player, world)
	require.NoError(t, err)
	require.NoError(t, da.Start(comp, player, world))
	for comp.State == gc.ActivityStateRunning {
		require.NoError(t, da.DoTurn(comp, player, world))
	}
	require.NoError(t, da.Finish(comp, player, world))

	require.True(t, world.ECS.Alive(hdd), "残数があるうちはエンティティが消えないべき")
	assert.Equal(t, 1, query.GetEntityCount(world, hdd), "1個だけ消費されるべき")
}

func TestDisassembleActivity_Finish_レベルアップでStatsChangedが付く(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player := newDisassembleTestPlayer(world)

	// 次の獲得でレベルアップする直前まで経験値を積んでおく
	mechanic := world.Components.Skills.Get(player).Get(gc.SkillMechanic)
	mechanic.Exp.Current = mechanic.Exp.Max - 1

	_, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	da := &DisassembleActivity{Target: crate}
	comp, err := da.BuildActivity(player, world)
	require.NoError(t, err)
	require.NoError(t, da.Start(comp, player, world))
	for comp.State == gc.ActivityStateRunning {
		require.NoError(t, da.DoTurn(comp, player, world))
	}
	require.NoError(t, da.Finish(comp, player, world))

	assert.GreaterOrEqual(t, mechanic.Value, 1, "機械スキルがレベルアップするべき")
	assert.True(t, world.Components.StatsChanged.Has(player),
		"レベルアップ時はステータス再計算マーカーが付くべき")
}

func TestDisassembleActivity_Validate_敵が隣接していると開始できない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player := newDisassembleTestPlayer(world)

	_, err := lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)
	spawnHostileAt(world, 9, 10)

	da := &DisassembleActivity{Target: crate}
	comp := &gc.Activity{Target: &crate}
	err = da.Validate(comp, player, world)
	require.Error(t, err)
	assert.ErrorContains(t, err, "敵")
}

func TestDisassembleActivity_DoTurn_敵が接近すると中断する(t *testing.T) {
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
	require.NoError(t, da.Validate(comp, player, world))
	require.NoError(t, da.Start(comp, player, world))
	require.NoError(t, da.DoTurn(comp, player, world))
	require.Equal(t, gc.ActivityStateRunning, comp.State, "敵がいなければ継続するべき")

	// 分解の途中で敵が隣接タイルまで近づいてきた
	spawnHostileAt(world, 10, 11)

	require.NoError(t, da.DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, "周囲に敵がいるため分解を中断", comp.CancelReason)
}

func TestAppendYields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stacks []lifecycle.YieldStack
		want   string
	}{
		{"空なら何も得られなかった", nil, "何も得られなかった"},
		{"1件は単独表記", []lifecycle.YieldStack{{Name: "鉄くず", Count: 2}}, "鉄くず x2 を得た"},
		{"複数件は読点で連結", []lifecycle.YieldStack{{Name: "鉄くず", Count: 2}, {Name: "硬木", Count: 1}}, "鉄くず x2、硬木 x1 を得た"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			store := query.GetGameLog(world)
			logger := gamelog.New(store)
			appendYields(logger, tt.stacks)
			logger.Log()

			recent := store.GetRecent(1)
			require.Len(t, recent, 1)
			assert.Equal(t, tt.want, recent[0])
		})
	}
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

	// 対象が消えた後のキャンセル処理でもエラーにならない
	require.NoError(t, da.Canceled(comp, player, world))
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
