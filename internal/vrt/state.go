package vrt

import (
	"image"
	"math/rand/v2"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/maingame"
	"github.com/kijimaD/ruins/internal/raw"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/require"
)

// States はステートのスライスをビルダー関数に変換するアダプタ
func States(states ...es.State[w.World]) func(w.World) []es.State[w.World] {
	return func(_ w.World) []es.State[w.World] { return states }
}

// RenderPNG はステートを構築し本番の renderer で描いてPNGを返す。比較はしない、画像保存用。
// screeneffect のポスト処理まで含めて実画面と同じ絵になる。
func RenderPNG(t *testing.T, buildStates func(w.World) []es.State[w.World]) []byte {
	t.Helper()
	return encodePNG(t, renderStates(t, buildStates))
}

// renderStates は world を作りステートを構築し、本番の MainGame.Draw で screen へ描いて NRGBA を返す。
func renderStates(t *testing.T, buildStates func(w.World) []es.State[w.World]) *image.NRGBA {
	t.Helper()
	world := InitReplayWorld(t)

	sm := SetupStateMachine(t, world, buildStates)
	game, err := maingame.NewMainGame(world, sm)
	require.NoError(t, err)
	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	game.Draw(screen)
	return captureScreen(screen)
}

// SetupStateMachine はステートを構築しレイアウト確定までフレームを回す。
func SetupStateMachine(t *testing.T, world w.World, buildStates func(w.World) []es.State[w.World]) es.StateMachine[w.World] {
	t.Helper()
	states := buildStates(world)
	require.NotEmpty(t, states, "at least one state is required")

	stateMachine, err := es.Init(states[0], world)
	require.NoError(t, err)
	require.NoError(t, stateMachine.Update(world))

	if len(states) > 1 {
		require.NoError(t, stateMachine.PushState(world, states[1:]...))
	}

	// レイアウト確定のためフレームを回す。警告なく失敗を握り潰すと不可解なテスト失敗になるので報告する
	for range 10 {
		require.NoError(t, stateMachine.Update(world), "layout warm-up failed")
	}
	return stateMachine
}

// InitReplayWorld は実プレイの再現に使うフルゲーム構成のワールドを固定シードで初期化する。
// テスト・ベンチ双方で使えるよう testing.TB を受ける。
//
// maingame.InitWorld はフォント読み込みなど並行安全でない構築を含むため loadMu で直列化する。
// 構築後の world は自前の独立フェイスを持つので、以降の設定と描画はロック無しで並列に走れる。
func InitReplayWorld(tb testing.TB) w.World {
	tb.Helper()

	cfg := &config.Config{Profile: config.ProfileDevelopment}
	cfg.ApplyProfileDefaults()
	cfg.LogLevel = "ignore"
	cfg.Seed = 12345
	cfg.RNG = rand.New(rand.NewPCG(cfg.Seed, 0))
	cfg.DisableAnimation = true
	// ポスト処理を切って撮る。スキャンラインの高周波パターンが無くなり、golden の
	// 差分がUIの実変化だけを映すようになる。トレランスを絞れる
	cfg.DisableScreenFilter = true
	require.NoError(tb, cfg.Validate())

	loadMu.Lock()
	world, err := maingame.InitWorld(cfg)
	loadMu.Unlock()
	require.NoError(tb, err)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(tb, err)

	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	if len(professions) > 0 {
		require.NoError(tb, gameaction.ApplyProfession(world, player, professions[0]))
	}

	for _, updater := range []w.Updater{
		&gs.StatsChangedSystem{},
		&gs.WeightDirtySystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			require.NoError(tb, sys.Update(world))
		}
	}
	return world
}
