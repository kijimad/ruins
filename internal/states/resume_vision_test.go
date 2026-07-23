package states_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 開始時の視界強制再計算は DungeonState.OnStart が両モードでまとめて立てる。serde は VisionState を
// 空で復元し、VisionSystem は現ステージが変わらないと再計算しないため、立てないと空の VisibleTiles の
// まま真っ暗になる。オーバーワールドと通常ダンジョンの両方でこの不変条件を固定する。

// TestDungeonResume_視界を強制再計算する は、通常ダンジョンのロード復帰で暗転しないことを固定する。
func TestDungeonResume_視界を強制再計算する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// OnStart は通常ダンジョンではタイトルエフェクトで UI リソースに触れるので用意する
	world.Resources.UIResources = vrt.SharedUIResources(t)
	// ロード直後を模す。現ステージは通常ダンジョンで、視界フラグは未設定
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(dungeon.DungeonDebug.Name(), 3)
	query.GetVisionState(world).NeedsForceUpdate = false

	st := &gs.DungeonState{Depth: 3, DefinitionName: dungeon.DungeonDebug.Name(), Resume: true}
	require.NoError(t, st.OnStart(world))

	assert.True(t, query.GetVisionState(world).NeedsForceUpdate, "通常ダンジョンの復帰は視界を強制再計算する")
}

// TestOverworldResume_視界を強制再計算する は、オーバーワールド開始/復帰で暗転しないことを固定する。
// 通常ダンジョンと同じく OnStart が開始時に視界の強制再計算を立てる。
func TestOverworldResume_視界を強制再計算する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	query.GetVisionState(world).NeedsForceUpdate = false

	// オーバーワールドの State は非公開フィールドを持つのでファクトリ経由で構成する。
	// Seamless 分岐は帯ドライバへ委譲して return するため、タイトルエフェクトには到達しない
	factory := gs.NewOverworldState(
		mapplanner.PlannerTypeOverworldField,
		dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3),
		&overworld.NewGameParams{RunSeed: 1},
	)
	state, err := factory()
	require.NoError(t, err)
	require.NoError(t, state.OnStart(world))

	assert.True(t, query.GetVisionState(world).NeedsForceUpdate, "オーバーワールド開始も視界を強制再計算する")
}
