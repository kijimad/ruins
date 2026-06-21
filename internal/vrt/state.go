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
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/require"
)

// States はステートのスライスをビルダー関数に変換するアダプタ
func States(states ...es.State[w.World]) func(w.World) []es.State[w.World] {
	return func(_ w.World) []es.State[w.World] { return states }
}

// AssertStateGolden はステートの描画結果をゴールデン画像と比較する。
// buildStatesはworld初期化後に呼ばれ、セットアップとステート構築を行う。
// セットアップが不要な場合はStatesアダプタを使う。
// GOLDIE_UPDATE=1 で実行するとゴールデン画像を更新する。
// ただし既存ゴールデンとのピクセル差分がトレランス内なら更新をスキップして、
// ebitenui の時間依存ノイズによる不要な差分を防ぐ
func AssertStateGolden(t *testing.T, buildStates func(w.World) []es.State[w.World]) {
	t.Helper()

	rendered := renderState(t, buildStates)
	pngData := encodePNG(t, rendered)
	assertPNGGolden(t, pngData)
}

// renderState はステートを描画してimage.NRGBAとして返す。
// RunTestMain 内で呼ぶ必要がある（ebitenコンテキストが必要）
func renderState(t *testing.T, buildStates func(w.World) []es.State[w.World]) *image.NRGBA {
	t.Helper()

	// World初期化はebitenui非依存なのでmutex外で並列実行できる
	world := InitVRTWorld(t)
	states := buildStates(world)
	require.NotEmpty(t, states, "ステートが1つ以上必要")

	// ebitenui内部の入力ハンドラが並行アクセス安全でないため直列化する。
	// ステート初期化もここで行うことで、mutex待機中にebitenui内部の
	// 時間ベースアニメーション（Caretブリンク等）が進行するのを防ぐ
	renderMu.Lock()
	defer renderMu.Unlock()

	stateMachine, err := es.Init(states[0], world)
	require.NoError(t, err)
	require.NoError(t, stateMachine.Update(world))

	if len(states) > 1 {
		require.NoError(t, stateMachine.PushState(world, states[1:]...))
	}

	// レイアウト確定のためフレームを回す
	for i := 0; i < 10; i++ {
		if err := stateMachine.Update(world); err != nil {
			break
		}
	}

	width, height := consts.MinGameWidth, consts.MinGameHeight
	screen := ebiten.NewImage(width, height)

	for _, state := range stateMachine.GetStates() {
		require.NoError(t, state.Draw(world, screen), "描画に失敗")
	}

	return captureScreen(screen)
}

// InitVRTWorld はVRT用のワールドを初期化する。固定シードで再現性を保証する
func InitVRTWorld(t *testing.T) w.World {
	t.Helper()

	cfg := &config.Config{Profile: config.ProfileDevelopment}
	cfg.ApplyProfileDefaults()
	cfg.LogLevel = "ignore"
	cfg.Seed = 12345
	cfg.RNG = rand.New(rand.NewPCG(cfg.Seed, 0))
	cfg.DisableAnimation = true
	require.NoError(t, cfg.Validate())

	world, err := maingame.InitWorld(cfg)
	require.NoError(t, err)

	player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	if len(professions) > 0 {
		require.NoError(t, worldhelper.ApplyProfession(world, player, professions[0]))
	}

	for _, updater := range []w.Updater{
		&gs.StatsChangedSystem{},
		&gs.InventoryChangedSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			require.NoError(t, sys.Update(world))
		}
	}

	return world
}
