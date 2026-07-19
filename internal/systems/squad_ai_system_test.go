package systems

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/aiinput"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAbilities() gc.Abilities {
	return gc.Abilities{
		Vitality:  gc.Ability{Base: 10},
		Strength:  gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7},
		Dexterity: gc.Ability{Base: 6},
		Agility:   gc.Ability{Base: 9},
		Defense:   gc.Ability{Base: 5},
	}
}

func TestSquadProcessor_護衛ポリシーでリーダーに追従する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	// 隊員をリーダーから遠くに配置する
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(20)
	memberGrid.Y = consts.Tile(20)

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialX := int(memberGrid.X)
	initialY := int(memberGrid.Y)

	// 隊員AI処理を実行する
	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// リーダーに近づいているはず
	finalGrid := world.Components.GridElement.Get(member)
	leaderGrid := world.Components.GridElement.Get(leader)

	initialDist := abs(initialX-int(leaderGrid.X)) + abs(initialY-int(leaderGrid.Y))
	finalDist := abs(int(finalGrid.X)-int(leaderGrid.X)) + abs(int(finalGrid.Y)-int(leaderGrid.Y))

	assert.Less(t, finalDist, initialDist, "リーダーに近づいているべき")
}

func TestSquadProcessor_護衛ポリシーでリーダー近くなら待機する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員B", testAbilities(), "player")
	require.NoError(t, err)

	// 隊員をリーダーの隣に配置する
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(11)
	memberGrid.Y = consts.Tile(10)

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// リーダーの近くにいるので位置が変わらない
	finalGrid := world.Components.GridElement.Get(member)
	assert.Equal(t, consts.Tile(11), finalGrid.X)
	assert.Equal(t, consts.Tile(10), finalGrid.Y)
}

func TestSquadProcessor_待機ポリシーで移動しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員C", testAbilities(), "player")
	require.NoError(t, err)

	// 待機ポリシーに変更する
	world.Components.SquadAI.Get(member).Movement = gc.SquadStationary

	// リーダーから遠くに配置する
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(20)
	memberGrid.Y = consts.Tile(20)

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// 待機ポリシーなので動かない
	finalGrid := world.Components.GridElement.Get(member)
	assert.Equal(t, consts.Tile(20), finalGrid.X)
	assert.Equal(t, consts.Tile(20), finalGrid.Y)
}

func TestSquadProcessor_死亡した隊員は処理されない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員D", testAbilities(), "player")
	require.NoError(t, err)

	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(20)
	memberGrid.Y = consts.Tile(20)

	// 死亡状態にする
	world.Components.Dead.Add(member, &gc.Dead{})

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// 死亡しているので動かない
	finalGrid := world.Components.GridElement.Get(member)
	assert.Equal(t, consts.Tile(20), finalGrid.X)
	assert.Equal(t, consts.Tile(20), finalGrid.Y)
}

func TestSquadProcessor_攻撃ポリシーで隣接した敵を攻撃する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員E", testAbilities(), "player")
	require.NoError(t, err)

	// 隊員をリーダーの隣に配置する
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(11)
	memberGrid.Y = consts.Tile(10)

	// 隊員の隣に敵を配置する
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "火の玉")
	require.NoError(t, err)

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	// 命中判定のRNGを固定して必ず命中させる。このテストの目的は攻撃ポリシーの検証であり命中ダイスの検証ではない
	world.Config.RNG = rand.New(rand.NewPCG(1, 0))

	// 敵の初期HPを記録する
	enemyHP := world.Components.HP.Get(enemy)
	initialHP := enemyHP.Current

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// 敵のHPが減っているべき
	assert.Less(t, enemyHP.Current, initialHP, "敵のHPが減っているべき")
}

func TestSquadProcessor_回避ポリシーで敵から離れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員F", testAbilities(), "player")
	require.NoError(t, err)

	// 回避ポリシーに変更する
	world.Components.SquadAI.Get(member).CombatCurrent = gc.CombatEvade

	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(12)
	memberGrid.Y = consts.Tile(10)

	// 隊員の近くに敵を配置する
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "火の玉")
	require.NoError(t, err)

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialX := int(memberGrid.X)

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// 敵から離れているはず（X座標が小さくなる方向）
	finalGrid := world.Components.GridElement.Get(member)
	assert.Less(t, int(finalGrid.X), initialX, "敵から離れているべき")
}

func TestSquadProcessor_プレイヤーを迂回してアイテムを拾いに行く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// レイアウト: アイテム(8,10) - プレイヤー(9,10) - 隊員(10,10)
	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 9, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員G", testAbilities(), "player")
	require.NoError(t, err)

	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(10)
	memberGrid.Y = consts.Tile(10)

	// アイテムをプレイヤーの向こう側に配置する
	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "テストアイテム"})
	world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(8), Y: consts.Tile(10)}})
	world.Components.LocationOnField.Add(item, &gc.LocationOnField{})

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialX := int(memberGrid.X)
	initialY := int(memberGrid.Y)

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	finalGrid := world.Components.GridElement.Get(member)
	finalX := int(finalGrid.X)
	finalY := int(finalGrid.Y)

	// 隊員がプレイヤーの位置(9,10)に突っ込まず、斜めに迂回して移動する
	moved := finalX != initialX || finalY != initialY
	assert.True(t, moved, "隊員がプレイヤーを迂回して移動すべき")
	assert.False(t, finalX == 9 && finalY == 10, "プレイヤーの位置には移動しない")
}

func TestSquadProcessor_HP低下で後退する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員H", testAbilities(), "player")
	require.NoError(t, err)

	// 隊員をリーダーから離す
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(15)
	memberGrid.Y = consts.Tile(10)

	// HPを閾値以下にする
	hp := world.Components.HP.Get(member)
	hp.Current = hp.Max * 25 / 100

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialX := int(memberGrid.X)

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	finalGrid := world.Components.GridElement.Get(member)
	leaderGrid := world.Components.GridElement.Get(leader)

	initialDist := abs(initialX-int(leaderGrid.X)) + abs(int(memberGrid.Y)-int(leaderGrid.Y))
	finalDist := abs(int(finalGrid.X)-int(leaderGrid.X)) + abs(int(finalGrid.Y)-int(leaderGrid.Y))

	assert.Less(t, finalDist, initialDist, "HP低下時はリーダーに向かって後退するべき")
}

func TestSquadProcessor_前衛ポリシーでリーダーから離れすぎると接近する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員I", testAbilities(), "player")
	require.NoError(t, err)

	// 前衛ポリシーに変更する
	world.Components.SquadAI.Get(member).Movement = gc.SquadVanguard

	// リーダーから最大距離を超えて配置する
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(20)
	memberGrid.Y = consts.Tile(10)

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialX := int(memberGrid.X)

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	finalGrid := world.Components.GridElement.Get(member)
	leaderGrid := world.Components.GridElement.Get(leader)

	initialDist := abs(initialX-int(leaderGrid.X)) + abs(int(memberGrid.Y)-int(leaderGrid.Y))
	finalDist := abs(int(finalGrid.X)-int(leaderGrid.X)) + abs(int(finalGrid.Y)-int(leaderGrid.Y))

	assert.Less(t, finalDist, initialDist, "前衛ポリシーでリーダーから離れすぎると接近するべき")
}

func TestSquadProcessor_前衛ポリシーでリーダー近くならランダム移動する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員J", testAbilities(), "player")
	require.NoError(t, err)

	// 前衛ポリシーに変更する
	world.Components.SquadAI.Get(member).Movement = gc.SquadVanguard

	// リーダーの近くに配置する（vanguardMaxDistance以内）
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(12)
	memberGrid.Y = consts.Tile(10)

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialX := int(memberGrid.X)
	initialY := int(memberGrid.Y)

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	finalGrid := world.Components.GridElement.Get(member)
	moved := int(finalGrid.X) != initialX || int(finalGrid.Y) != initialY
	assert.True(t, moved, "前衛ポリシーでリーダー近くならランダム移動するべき")
}

func TestSquadProcessor_巡回ポリシーで移動する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員K", testAbilities(), "player")
	require.NoError(t, err)

	// 巡回ポリシーに変更する
	world.Components.SquadAI.Get(member).Movement = gc.SquadPatrol

	// リーダーの近くに配置する
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(12)
	memberGrid.Y = consts.Tile(10)

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialX := int(memberGrid.X)
	initialY := int(memberGrid.Y)

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	finalGrid := world.Components.GridElement.Get(member)
	moved := int(finalGrid.X) != initialX || int(finalGrid.Y) != initialY
	assert.True(t, moved, "巡回ポリシーで移動するべき")
}

func TestSquadProcessor_CombatIgnoreで戦闘しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員L", testAbilities(), "player")
	require.NoError(t, err)

	// CombatIgnoreに変更する
	squad := world.Components.SquadAI.Get(member)
	squad.CombatDefault = gc.CombatIgnore
	squad.CombatCurrent = gc.CombatIgnore

	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(11)
	memberGrid.Y = consts.Tile(10)

	// 隣接する敵を配置する。敵からの攻撃を防ぐためCombatIgnoreにする
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "火の玉")
	require.NoError(t, err)
	enemySolo := world.Components.SoloAI.Get(enemy)
	enemySolo.CombatDefault = gc.CombatIgnore
	enemySolo.CombatCurrent = gc.CombatIgnore

	// 探索済みタイルを設定する
	dungeon := query.GetDungeon(world)
	for x := range 50 {
		for y := range 50 {
			dungeon.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}

	initialHP := world.Components.HP.Get(enemy).Current

	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// Getのポインタは構造変更で失効するため、検証時に再取得する
	assert.Equal(t, initialHP, world.Components.HP.Get(enemy).Current, "CombatIgnoreでは敵を攻撃しない")
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
