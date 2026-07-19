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

// AssertStateGolden はステートの描画結果をゴールデン画像と比較する。
// buildStatesはworld初期化後に呼ばれ、セットアップとステート構築を行う。
// セットアップが不要な場合はStatesアダプタを使う。
// GOLDIE_UPDATE=1 で実行するとゴールデン画像を更新する。
// ただし既存ゴールデンとのピクセル差分がトレランス内なら更新をスキップして、
// ebitenui の時間依存ノイズによる不要な差分を防ぐ
func AssertStateGolden(t *testing.T, buildStates func(w.World) []es.State[w.World]) {
	t.Helper()
	assertPNGGolden(t, RenderStatePNG(t, buildStates))
}

// RenderStatePNG はステートを描画してPNGバイト列として返す。
// アサーションは行わず、画像の保存用途で使用する
func RenderStatePNG(t *testing.T, buildStates func(w.World) []es.State[w.World]) []byte {
	t.Helper()
	rendered := renderState(t, buildStates)
	return encodePNG(t, rendered)
}

// renderState はステートを描画してimage.NRGBAとして返す。
// RunTestMain 内で呼ぶ必要がある（ebitenコンテキストが必要）
func renderState(t *testing.T, buildStates func(w.World) []es.State[w.World]) *image.NRGBA {
	t.Helper()

	// World初期化・状態構築・描画はいずれも ebitenui のグローバル描画状態に触れて並行アクセス安全でない。
	// これらを renderMu で直列化する。World 初期化は InitVRTWorld が内部で同じ renderMu を取るので、
	// ここでは構築から描画までを別区間としてロックする。両区間とも renderMu なので ebitenui グローバルへの
	// 同時アクセスは起きない。mutex待機中に ebitenui の時間ベースアニメーション（Caretブリンク等）が進むのも防ぐ
	world := InitVRTWorld(t)

	renderMu.Lock()
	defer renderMu.Unlock()

	states := buildStates(world)
	require.NotEmpty(t, states, "ステートが1つ以上必要")

	stateMachine, err := es.Init(states[0], world)
	require.NoError(t, err)
	require.NoError(t, stateMachine.Update(world))

	if len(states) > 1 {
		require.NoError(t, stateMachine.PushState(world, states[1:]...))
	}

	// レイアウト確定のためフレームを回す
	for range 10 {
		if err := stateMachine.Update(world); err != nil {
			break
		}
	}

	width, height := consts.GameWidth, consts.GameHeight
	screen := ebiten.NewImage(width, height)

	for _, state := range stateMachine.GetStates() {
		require.NoError(t, state.Draw(world, screen), "描画に失敗")
	}

	return captureScreen(screen)
}

// InitVRTWorld はVRT用のワールドを初期化する。固定シードで再現性を保証する。
// テスト・ベンチ双方から使えるよう testing.TB を受ける。
//
// maingame.InitWorld 経由で ebitenui のグローバルな NineSlice キャッシュを触るため renderMu で
// 直列化する。触らないと並列ゴールデンテストの初期化と描画が同時にこのキャッシュへアクセスして
// data race になる。renderMu は非再入なので、renderState はこの関数を描画ロックの外側で呼ぶ。
func InitVRTWorld(tb testing.TB) w.World {
	tb.Helper()
	renderMu.Lock()
	defer renderMu.Unlock()

	cfg := &config.Config{Profile: config.ProfileDevelopment}
	cfg.ApplyProfileDefaults()
	cfg.LogLevel = "ignore"
	cfg.Seed = 12345
	cfg.RNG = rand.New(rand.NewPCG(cfg.Seed, 0))
	cfg.DisableAnimation = true
	require.NoError(tb, cfg.Validate())

	world, err := maingame.InitWorld(cfg)
	require.NoError(tb, err)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
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
