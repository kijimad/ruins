package query

import (
	"testing"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDungeon(t *testing.T) {
	t.Parallel()

	t.Run("InitWorldで生成されたDungeonを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		d := GetDungeon(world)
		require.NotNil(t, d)
	})

	t.Run("SetDungeonで設定した値を取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		newDungeon := &gc.Dungeon{CurrentStage: gc.NewDungeonStage("テスト遺跡", 3)}
		SetDungeon(world, newDungeon)

		d := GetDungeon(world)
		require.NotNil(t, d)
		assert.Equal(t, 3, d.CurrentStage.Depth)
	})

	t.Run("SetDungeonでnilを設定するとGetDungeonはnilを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		SetDungeon(world, nil)

		d := GetDungeon(world)
		assert.Nil(t, d)
	})

	t.Run("コンポーネントを持たない状態からSetDungeonすると新たに追加する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 先にnilで設定し、Dungeonコンポーネントを取り除いた状態を作る
		SetDungeon(world, nil)

		newDungeon := &gc.Dungeon{CurrentStage: gc.NewDungeonStage("追加テスト遺跡", 2)}
		SetDungeon(world, newDungeon)

		d := GetDungeon(world)
		require.NotNil(t, d, "コンポーネントが無い状態からでも追加できる")
		assert.Equal(t, 2, d.CurrentStage.Depth)
	})
}

// TestIsOnOverworld は現在地判定を検証する。
func TestIsOnOverworld(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := GetDungeon(world)

	// オーバーワールドの StageField に帯データを持たせる。以後この帯データの有無で判定する
	d.CurrentStage = gc.NewOverworldStage()
	EnsureSeamlessBand(world)
	assert.True(t, IsOnOverworld(world), "現ステージが帯データを持てば真")

	// 遺跡滞在中。現ステージの StageField は帯データを持たないので偽。帯データはオーバーワールドの
	// StageField にしか無く、退避されて現ステージから外れる
	d.CurrentStage = gc.NewDungeonStage("テスト遺跡", 1)
	assert.False(t, IsOnOverworld(world), "現ステージが帯データを持たなければ偽")
}

// TestGetWeaponSelection は武器選択シングルトンの初期値と更新を検証する。
func TestGetWeaponSelection(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ws := GetWeaponSelection(world)
	require.NotNil(t, ws)
	assert.Equal(t, 1, ws.Slot, "初期武器スロットは1")

	ws.Slot = 3
	assert.Equal(t, 3, GetWeaponSelection(world).Slot, "更新がシングルトンに反映される")
}

func TestGetRunStats_InitWorldで初期状態を取得できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	stats := GetRunStats(world)
	require.NotNil(t, stats)
	assert.Equal(t, 0, stats.EnemiesKilled)

	stats.EnemiesKilled = 5
	assert.Equal(t, 5, GetRunStats(world).EnemiesKilled, "更新がシングルトンに反映される")
}

func TestGetSeamlessBand_帯データの有無で結果が変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := GetDungeon(world)
	d.CurrentStage = gc.NewOverworldStage()

	assert.Nil(t, GetSeamlessBand(world), "帯データを確保する前はnil")

	EnsureSeamlessBand(world)
	assert.NotNil(t, GetSeamlessBand(world), "確保した後は取得できる")
}

func TestEnsureStageField_同じキーなら既存のエンティティを再利用する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	key := gc.NewDungeonStage("テスト遺跡", 1)

	first := EnsureStageField(world, key)
	require.NotNil(t, first)
	first.Level.TileWidth = consts.Tile(99)

	second := EnsureStageField(world, key)
	assert.Equal(t, consts.Tile(99), second.Level.TileWidth, "2回目は既存のエンティティを返す")
}

// TestGetPlayTime_InitWorldで生成されたシングルトンを取得できる は、InitWorld が
// シングルトンエンティティに PlayTime を確保済みで、GetPlayTime がそれを取得することを固定する。
func TestGetPlayTime_InitWorldで生成されたシングルトンを取得できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	pt := GetPlayTime(world)

	require.NotNil(t, pt)
	assert.Equal(t, time.Duration(0), pt.Elapsed(), "開始直後は経過時間0")
}

func TestGetCurrentStageField_未生成のステージはnilを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := GetDungeon(world)
	d.CurrentStage = gc.NewDungeonStage("未生成の遺跡", 1)

	assert.Nil(t, GetCurrentStageField(world), "StageFieldを生成していないステージはnil")
}

// TestGetGameLog_未初期化ならnilを返す は、GameLog シングルトンが無い状態での読みを固定する。
func TestGetGameLog_未初期化ならnilを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Components.GameLog.Remove(world.Resources.SingletonEntity)

	assert.Nil(t, GetGameLog(world))
}

// TestGetSpatialIndex_コンポーネントが無ければnil は、SpatialIndex シングルトンを
// 持たない世界では構築を試みず直ちに nil を返すことを固定する。
func TestGetSpatialIndex_コンポーネントが無ければnil(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Components.SpatialIndex.Remove(world.Resources.SingletonEntity)

	assert.Nil(t, GetSpatialIndex(world))
}

// TestGetSpatialIndex_StageField未生成のステージでは構築できずnil は、buildSpatialIndex が
// field==nil で早期returnしMapWidth/Heightが0のままとなり、GetSpatialIndexがnilへ落ちることを固定する。
func TestGetSpatialIndex_StageField未生成のステージでは構築できずnil(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	GetDungeon(world).CurrentStage = gc.NewDungeonStage("未生成の遺跡", 1)

	assert.Nil(t, GetSpatialIndex(world))
}

// TestUpdateCharacterPositionInIndex_コンポーネントが無ければ何もしない は、
// 索引シングルトンが無い状態で呼んでもpanicせず何もしないことを固定する。
func TestUpdateCharacterPositionInIndex_コンポーネントが無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Components.SpatialIndex.Remove(world.Resources.SingletonEntity)

	e := world.ECS.NewEntity()
	assert.NotPanics(t, func() {
		UpdateCharacterPositionInIndex(world, e, consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 1, Y: 0})
	})
}
