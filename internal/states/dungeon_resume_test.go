package states_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDungeonResume_視界を強制再計算する は、ダンジョンのロード復帰で真っ暗にならないことを固定する。
// serde は VisionState を空にして復元するため、復帰の OnStart で強制再計算を立てないと、
// 現ステージが保存前と同じで VisionSystem のフロア変化検知も働かず、空の VisibleTiles のまま暗転する。
func TestDungeonResume_視界を強制再計算する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// OnStart はタイトルエフェクトで UI リソースに触れるので用意する
	world.Resources.UIResources = vrt.SharedUIResources(t)
	// ロード直後を模す。現ステージは通常ダンジョンで、視界フラグは未設定
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(dungeon.DungeonDebug.Name(), 3)
	query.GetVisionState(world).NeedsForceUpdate = false

	st := &gs.DungeonState{Depth: 3, DefinitionName: dungeon.DungeonDebug.Name(), Resume: true}
	require.NoError(t, st.OnStart(world))

	assert.True(t, query.GetVisionState(world).NeedsForceUpdate, "ロード復帰時は視界を強制再計算する")
}
