package maingame

import (
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMainGame_フィルタ有効時はレトロフィルタ込みで構築する は、
// DisableScreenFilter が false のときに NewMainGame がエラーなく構築でき、
// 組み込まれたレトロフィルタ込みの pipeline で Draw がパニックしないことを確認する。
func TestNewMainGame_フィルタ有効時はレトロフィルタ込みで構築する(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Profile: config.ProfileDevelopment}
	cfg.ApplyProfileDefaults()
	cfg.DisableScreenFilter = false
	world, err := InitWorld(cfg)
	require.NoError(t, err)

	stateMachine, err := es.Init(&gs.MainMenuState{}, world)
	require.NoError(t, err)

	game, err := NewMainGame(world, stateMachine)
	require.NoError(t, err)
	require.NotNil(t, game)

	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	assert.NotPanics(t, func() {
		game.Draw(screen)
	})
}

// TestNewMainGame_フィルタ無効時はフィルタなしで構築する は、
// DisableScreenFilter が true のとき、レトロフィルタを組まずにエラーなく構築でき、
// フィルタなしの pipeline で Draw がパニックしないことを確認する。
func TestNewMainGame_フィルタ無効時はフィルタなしで構築する(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Profile: config.ProfileDevelopment}
	cfg.ApplyProfileDefaults()
	cfg.DisableScreenFilter = true
	world, err := InitWorld(cfg)
	require.NoError(t, err)

	stateMachine, err := es.Init(&gs.MainMenuState{}, world)
	require.NoError(t, err)

	game, err := NewMainGame(world, stateMachine)
	require.NoError(t, err)
	require.NotNil(t, game)

	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	assert.NotPanics(t, func() {
		game.Draw(screen)
	})
}

// TestGetPerformanceInfo_GC実施後は経過時間を表示する は、
// runtime.GC() 実施後に LastGC が "N/A" ではなく経過秒数として表示され、
// 期待するフィールドが期待する順序で並ぶことを確認する。
func TestGetPerformanceInfo_GC実施後は経過時間を表示する(t *testing.T) {
	t.Parallel()

	runtime.GC()

	info := getPerformanceInfo()

	assert.Regexp(t,
		`^FPS: .+\nAlloc: .+MB\nHeapInuse: .+MB\nStackInuse: .+MB\nSys: .+MB\nNextGC: .+MB\nTotalAlloc: .+MB\nMallocs: \d+\nFrees: \d+\nGC: \d+\nLastGC: \d+\.\d\ds\nPauseTotalNs: .+ms\nGoroutines: \d+\n$`,
		info,
	)
	assert.NotContains(t, info, "N/A", "直前にGCしているのでLastGCはN/Aにならない")
}
