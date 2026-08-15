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
	// InitVRTWorld が内部で WithUILock を取るので、構築から描画まではもう1つの WithUILock 区間にする。
	// WithUILock は非再入なのでネストさせず2区間に分ける。両区間とも同じロックなので ebitenui グローバルへの
	// 同時アクセスは起きない。mutex待機中に ebitenui の時間ベースアニメーション（Caretブリンク等）が進むのも防ぐ
	world := InitVRTWorld(t)

	var out *image.NRGBA
	WithUILock(func() {
		stateMachine := setupStateMachine(t, world, buildStates)

		width, height := consts.GameWidth, consts.GameHeight
		screen := ebiten.NewImage(width, height)

		for _, state := range stateMachine.GetStates() {
			require.NoError(t, state.Draw(world, screen), "failed to draw")
		}

		out = captureScreen(screen)
	})
	return out
}

// setupStateMachine はステートを構築しレイアウト確定までフレームを回す。
// ebitenui グローバルに触れるため WithUILock 区間から呼ぶ。
func setupStateMachine(t *testing.T, world w.World, buildStates func(w.World) []es.State[w.World]) es.StateMachine[w.World] {
	t.Helper()
	states := buildStates(world)
	require.NotEmpty(t, states, "at least one state is required")

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
	return stateMachine
}

// BuildWorld はステートを構築し、レイアウト確定までフレームを回した world を返す。
// 描画はしない。Render3DSystem などステート外のレンダラで描いたり world を検査するのに使う。
// ステート構築は ebitenui グローバルに触れるため WithUILock 内で行う。
func BuildWorld(t *testing.T, buildStates func(w.World) []es.State[w.World]) w.World {
	t.Helper()
	world := InitVRTWorld(t)
	WithUILock(func() {
		setupStateMachine(t, world, buildStates)
	})
	return world
}

// RenderWorldPNG はステートで world を構築し、任意の描画関数でオフスクリーンへ描いてPNGを返す。
// 2Dステートの描画に依らず Render3DSystem のようなレンダラを画像化するVRTで使う。アサーションはしない。
func RenderWorldPNG(t *testing.T, buildStates func(w.World) []es.State[w.World], draw func(world w.World, screen *ebiten.Image)) []byte {
	t.Helper()
	world := InitVRTWorld(t)
	var out *image.NRGBA
	WithUILock(func() {
		setupStateMachine(t, world, buildStates)
		screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
		draw(world, screen)
		out = captureScreen(screen)
	})
	return encodePNG(t, out)
}

// InitVRTWorld はVRT用のワールドを初期化する。固定シードで再現性を保証する。
// テスト・ベンチ双方から使えるよう testing.TB を受ける。
//
// maingame.InitWorld 経由で ebitenui のグローバルな NineSlice キャッシュを触るため WithUILock で
// 直列化する。触らないと並列ゴールデンテストの初期化と描画が同時にこのキャッシュへアクセスして
// data race になる。WithUILock は非再入なので、renderState はこの関数を描画の WithUILock 区間の外側で呼ぶ。
func InitVRTWorld(tb testing.TB) w.World {
	tb.Helper()

	var world w.World
	WithUILock(func() {
		cfg := &config.Config{Profile: config.ProfileDevelopment}
		cfg.ApplyProfileDefaults()
		cfg.LogLevel = "ignore"
		cfg.Seed = 12345
		cfg.RNG = rand.New(rand.NewPCG(cfg.Seed, 0))
		cfg.DisableAnimation = true
		require.NoError(tb, cfg.Validate())

		w2, err := maingame.InitWorld(cfg)
		require.NoError(tb, err)
		world = w2

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
	})
	return world
}
