package query_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSpatialIndex_キャラクターはBlockPassに含まれない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	query.InvalidateSpatialIndex(world)
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)

	memberGrid := world.Components.GridElement.Get(member)
	leaderGrid := world.Components.GridElement.Get(leader)

	assert.False(t, si.BlockPass[*leaderGrid], "プレイヤーはBlockPassに含まれない")
	assert.False(t, si.BlockPass[*memberGrid], "隊員はBlockPassに含まれない")

	_, leaderInChars := si.Characters[*leaderGrid]
	assert.True(t, leaderInChars, "プレイヤーはCharactersに含まれる")

	_, memberInChars := si.Characters[*memberGrid]
	assert.True(t, memberInChars, "隊員はCharactersに含まれる")
}

func TestBuildSpatialIndex_退避中エンティティは索引に含まれない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 現ステージの敵。索引に載るべき
	activePos := consts.Coord[consts.Tile]{X: 5, Y: 5}
	active := world.ECS.NewEntity()
	world.Components.GridElement.Add(active, &gc.GridElement{Coord: activePos})
	world.Components.SoloAI.Add(active, &gc.SoloAI{})

	// 退避中ステージの敵。索引に載らないべき
	suspendedPos := consts.Coord[consts.Tile]{X: 6, Y: 6}
	suspendedChar := world.ECS.NewEntity()
	world.Components.GridElement.Add(suspendedChar, &gc.GridElement{Coord: suspendedPos})
	world.Components.SoloAI.Add(suspendedChar, &gc.SoloAI{})
	world.Components.Suspended.Add(suspendedChar, &gc.Suspended{})

	// 退避中の障害物。BlockPass 索引に載らないべき
	wallPos := consts.Coord[consts.Tile]{X: 7, Y: 7}
	wall := world.ECS.NewEntity()
	world.Components.GridElement.Add(wall, &gc.GridElement{Coord: wallPos})
	world.Components.BlockPass.Add(wall, &gc.BlockPass{})
	world.Components.Suspended.Add(wall, &gc.Suspended{})

	query.InvalidateSpatialIndex(world)
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)

	_, activeIn := si.Characters[gc.GridElement{Coord: activePos}]
	assert.True(t, activeIn, "現ステージの敵は索引に含まれる")
	_, suspendedIn := si.Characters[gc.GridElement{Coord: suspendedPos}]
	assert.False(t, suspendedIn, "退避中の敵は索引に含まれない")
	assert.False(t, si.BlockPass[gc.GridElement{Coord: wallPos}], "退避中の障害物は BlockPass に含まれない")
}

func TestUpdateCharacterPositionInIndex_増分更新でBuiltを保つ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)
	require.True(t, si.Built)

	// (10,10) -> (11,10) へ増分更新する（無効化しない）
	query.UpdateCharacterPositionInIndex(world, player, consts.Coord[consts.Tile]{X: 10, Y: 10}, consts.Coord[consts.Tile]{X: 11, Y: 10})

	assert.True(t, si.Built, "増分更新は再構築を起こさずBuiltを保つ")
	_, oldOccupied := si.CharacterAt(consts.Coord[consts.Tile]{X: 10, Y: 10})
	assert.False(t, oldOccupied, "旧タイルは空になる")
	got, newOccupied := si.CharacterAt(consts.Coord[consts.Tile]{X: 11, Y: 10})
	assert.True(t, newOccupied, "新タイルにキャラがいる")
	assert.Equal(t, player, got, "新タイルは移動したエンティティ")
}

func TestUpdateCharacterPositionInIndex_入れ替えは順序非依存(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnSquadMember(world, player, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)

	pg := world.Components.GridElement.Get(player)
	mg := world.Components.GridElement.Get(member)
	pPos := pg.Coord
	mPos := mg.Coord
	require.NotEqual(t, pPos, mPos, "隊員は別タイルにスポーンする")

	// 位置入れ替え：player を member タイルへ、member を player タイルへ（actor 先で更新）。
	// MoveCharacter が「from が自分自身のときだけ削除」するため、順序に関わらず最終状態が正しくなる
	query.UpdateCharacterPositionInIndex(world, player, pPos, mPos)
	query.UpdateCharacterPositionInIndex(world, member, mPos, pPos)

	gotAtMemberTile, ok1 := si.CharacterAt(mPos)
	assert.True(t, ok1, "元memberタイルは埋まっている")
	assert.Equal(t, player, gotAtMemberTile, "元memberタイルにplayerが入る")
	gotAtPlayerTile, ok2 := si.CharacterAt(pPos)
	assert.True(t, ok2, "元playerタイルは埋まっている")
	assert.Equal(t, member, gotAtPlayerTile, "元playerタイルにmemberが入る")
}

func TestSpatialIndex_移動で再構築チャーンが起きない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)
	buildsAfterInit := si.BuildCount
	require.Positive(t, buildsAfterInit, "初回アクセスで1回は構築される")

	// プレイヤーを数回移動する。増分更新されるべきで、再構築（チャーン）は起きない
	const steps = 5
	for range steps {
		grid := world.Components.GridElement.Get(player)
		dest := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: grid.X + 1, Y: grid.Y}}
		_, err := activity.Execute(&activity.MoveActivity{Destination: dest}, player, world)
		require.NoError(t, err)
	}

	assert.Equal(t, buildsAfterInit, si.BuildCount,
		"移動は増分更新され、空間インデックスの再構築チャーンを起こさない（旧実装なら移動数だけ増える）")

	// 位置もインデックスに正しく反映されている
	got, ok := si.CharacterAt(consts.Coord[consts.Tile]{X: 15, Y: 10})
	assert.True(t, ok, "移動後の位置がインデックスに反映されている")
	assert.Equal(t, player, got, "移動先タイルはプレイヤー")
}

func TestInvalidateSpatialIndex_全マップがクリアされる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)
	assert.True(t, si.Built)
	assert.NotNil(t, si.Characters)

	query.InvalidateSpatialIndex(world)

	si2 := query.GetSingleton[gc.SpatialIndex](world, world.Components.SpatialIndex)
	assert.False(t, si2.Built)
	assert.Nil(t, si2.Characters)
	assert.Nil(t, si2.BlockPass)
	assert.Nil(t, si2.PlayerEntity)
}
