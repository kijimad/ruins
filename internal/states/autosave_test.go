package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
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

func TestAutoSaver_saveがファイルを書く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	m := newTempManager(t)
	a := &autoSaver{enabled: true, manager: m}

	a.save(world)

	assert.Equal(t, 1, countAutoSaves(t, m), "save でオートセーブファイルが1つできる")
}

func TestAutoSaver_無効なら何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	a := &autoSaver{enabled: false}

	assert.NotPanics(t, func() { a.save(world) }, "no-op で panic しない")
}

func TestNewAutoSaver_セーブ無効なら無効化する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.SaveLoadEnabled = false
	world.Resources.Config.DisableAutoSave = false // SaveLoadEnabled だけを要因に絞る

	a := newAutoSaver(world)

	assert.False(t, a.enabled, "セーブ無効では有効化しない")
}

func TestNewAutoSaver_再生時は無効化する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.SaveLoadEnabled = true
	world.Resources.Config.DisableAutoSave = true // DisableAutoSave だけを要因に絞る

	a := newAutoSaver(world)

	assert.False(t, a.enabled, "再生では有効化せず副作用を出さない")
}

func TestDungeonState_入眠でオートセーブする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	// 過労にして入眠可能にする。SpawnPlayer が既に Fatigue を持つので値を書き換える
	fat := world.Components.Fatigue.Get(player)
	require.NotNil(t, fat, "SpawnPlayer は Fatigue を持つ")
	fat.Current = fat.Max
	// テスト world の既定気温は就寝可能域を1度下回るので、ステージ基準温度を上げて域内にする
	if field := query.GetCurrentStageField(world); field != nil {
		field.BaseTemp += 20
	}
	require.True(t, activity.EvaluateSleepConditions(world, player).CanSleep(), "前提: 入眠可能")

	m := newTempManager(t)
	st := &DungeonState{autoSave: &autoSaver{enabled: true, manager: m}}

	_, choices := st.sleepConfirmChoices(world)
	var sleepRun func(w.World) (es.Transition[w.World], error)
	for _, c := range choices {
		if c.Label == query.T(world, "Sleep") {
			sleepRun = c.Run
		}
	}
	require.NotNil(t, sleepRun, "Sleep 選択肢がある")

	_, rerr := sleepRun(world)
	require.NoError(t, rerr)

	assert.Equal(t, 1, countAutoSaves(t, m), "入眠でオートセーブが1つできる")
}
