package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTempManager は一時ディレクトリを保存先にした SerializationManager を返す。
func newTempManager(t *testing.T) *save.SerializationManager {
	t.Helper()
	m, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	return m
}

// countAutoSaves はオートセーブスロット数を返す。
func countAutoSaves(t *testing.T, sm *save.SerializationManager) int {
	t.Helper()
	list, err := sm.ListAutoSaves()
	require.NoError(t, err)
	return len(list)
}

func TestAutoSaver_新規開始で初回保存する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	ts := query.GetTurnState(world)
	ts.TurnNumber = 1
	ts.LastAutoSaveTurn = 0 // 未オートセーブ。新規開始直後を表す
	m := newTempManager(t)
	a := &autoSaver{manager: m}

	a.maybeSave(world)

	assert.Equal(t, consts.Turn(1), ts.LastAutoSaveTurn, "保存後は現在ターンを刻む")
	assert.Equal(t, 1, countAutoSaves(t, m), "初回保存でファイルが1つできる")
}

func TestAutoSaver_間隔未満では保存しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	ts := query.GetTurnState(world)
	ts.TurnNumber = 100
	ts.LastAutoSaveTurn = 1 // 経過99 < 150
	m := newTempManager(t)
	a := &autoSaver{manager: m}

	a.maybeSave(world)

	assert.Equal(t, consts.Turn(1), ts.LastAutoSaveTurn, "保存しないので刻みは変わらない")
	assert.Equal(t, 0, countAutoSaves(t, m))
}

func TestAutoSaver_間隔到達で保存する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	ts := query.GetTurnState(world)
	ts.TurnNumber = 151
	ts.LastAutoSaveTurn = 1 // 経過150 >= 150
	m := newTempManager(t)
	a := &autoSaver{manager: m}

	a.maybeSave(world)

	assert.Equal(t, consts.Turn(151), ts.LastAutoSaveTurn)
	assert.Equal(t, 1, countAutoSaves(t, m))
}

func TestAutoSaver_継続アクティビティ中は抑止する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	world.Components.Activity.Add(player, &gc.Activity{BehaviorName: gc.BehaviorRest, State: gc.ActivityStateRunning})
	require.True(t, query.HasActivity(world, player), "前提: 継続アクティビティ中")

	ts := query.GetTurnState(world)
	ts.TurnNumber = 300
	ts.LastAutoSaveTurn = 0 // 抑止が無ければ発火する条件
	m := newTempManager(t)
	a := &autoSaver{manager: m}

	a.maybeSave(world)

	assert.Equal(t, consts.Turn(0), ts.LastAutoSaveTurn, "継続中は保存せず刻みも変えない")
	assert.Equal(t, 0, countAutoSaves(t, m))
}

func TestAutoSaver_セーブ無効なら何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.SaveLoadEnabled = false

	a := newAutoSaver(world)
	require.Nil(t, a.manager, "セーブ無効ではマネージャを持たない")

	ts := query.GetTurnState(world)
	ts.TurnNumber = 1000
	ts.LastAutoSaveTurn = 0

	assert.NotPanics(t, func() { a.maybeSave(world) }, "no-op で panic しない")
	assert.Equal(t, consts.Turn(0), ts.LastAutoSaveTurn)
}
