package vrt

import (
	"encoding/json"
	"image"
	"math/rand/v2"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sebdah/goldie/v2"

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

// SnapshotFunc はステートの論理内容をビューモデルとして返す。対応しないステートは ok=false を返す。
// プロダクションの vrt 非依存を保つため、実装はステート側パッケージのテストが型スイッチで持つ
type SnapshotFunc func(w.World, es.State[w.World]) (any, bool)

// AssertStateGolden はステートスタックの決定的な論理内容をゴールデンと突き合わせる。
// buildStates は world 初期化後に呼ばれ、セットアップとステート構築を行う。セットアップが不要な場合は
// States アダプタを使う。
//
// snapshot がビューモデルを返すステートはそれを JSON にして .json ゴールデンへ。返すステートが無ければ
// world を WorldSnapshot のテキストとして .txt ゴールデンへ。どちらも無い純UIメニューはテキスト assert を
// しない。画像はピクセル比較せず、Draw を毎回実行して描画のパニックやエラーだけを smoke check として
// 検出する。ピクセルを比較しないので描画の非決定性はフレークにならない。画像は目視用に GOLDIE_UPDATE
// 時のみ保存する。
func AssertStateGolden(t *testing.T, snapshot SnapshotFunc, buildStates func(w.World) []es.State[w.World]) {
	t.Helper()
	world := InitVRTWorld(t)

	renderMu.Lock()
	defer renderMu.Unlock()

	sm := driveStates(t, world, buildStates)

	if content, suffix, ok := snapshotStack(t, world, snapshot, sm.GetStates()); ok {
		goldie.New(t, goldie.WithNameSuffix(suffix)).Assert(t, t.Name(), []byte(content))
	}

	// Draw を毎回実行し、パニックやエラーが起きれば drawStates 内の require で落とす。純UIメニューは
	// テキスト assert を持たないが、この smoke check で描画の破綻だけは検出する。画像の保存は更新時のみ。
	img := drawStates(t, world, sm)
	if isGoldieUpdate() {
		writeImageArtifact(t, encodePNG(t, img))
	}
}

// snapshotStack はスタックの決定的な論理内容とゴールデンの拡張子を返す。snapshot がビューモデルを返す
// ステートがあればそれらを JSON へ直列化する。ビューモデルを直接直列化するので手書きの整形が要らず、
// フィールド追加は自動でゴールデンに現れる。ビューモデルが無ければ world を WorldSnapshot にする。
// マップもビューモデルも無い純UIメニューは ok=false を返し、呼び出し側はテキスト assert をしない
func snapshotStack(t *testing.T, world w.World, snapshot SnapshotFunc, states []es.State[w.World]) (string, string, bool) {
	t.Helper()
	var parts []any
	for _, s := range states {
		if v, ok := snapshot(world, s); ok {
			parts = append(parts, v)
		}
	}
	if len(parts) > 0 {
		payload := parts[0]
		if len(parts) > 1 {
			payload = parts
		}
		b, err := json.MarshalIndent(payload, "", "  ")
		require.NoError(t, err, "ビューモデルの JSON 直列化に失敗")
		return string(b) + "\n", ".json", true
	}
	if snap := SnapshotWorld(world); len(snap.Grid) > 0 || len(snap.Entities) > 0 {
		return snap.String(), ".txt", true
	}
	return "", "", false
}

// RenderStatePNG はステートを描画してPNGバイト列として返す。アサーションは行わず、画像の保存用途で使う。
func RenderStatePNG(t *testing.T, buildStates func(w.World) []es.State[w.World]) []byte {
	t.Helper()
	return encodePNG(t, renderState(t, buildStates))
}

// renderState はステートを構築・駆動して描画し image.NRGBA として返す。RunTestMain 内で呼ぶ必要がある。
func renderState(t *testing.T, buildStates func(w.World) []es.State[w.World]) *image.NRGBA {
	t.Helper()
	world := InitVRTWorld(t)
	renderMu.Lock()
	defer renderMu.Unlock()
	return drawStates(t, world, driveStates(t, world, buildStates))
}

// driveStates は buildStates からステートを構築し、レイアウト確定までフレームを回す。描画はしない。
// ebitenui のグローバル描画状態と world 構築に触れるので、renderMu を保持した状態で呼ぶ。
func driveStates(t *testing.T, world w.World, buildStates func(w.World) []es.State[w.World]) *es.StateMachine[w.World] {
	t.Helper()
	states := buildStates(world)
	require.NotEmpty(t, states, "ステートが1つ以上必要")

	sm, err := es.Init(states[0], world)
	require.NoError(t, err)
	require.NoError(t, sm.Update(world))

	if len(states) > 1 {
		require.NoError(t, sm.PushState(world, states[1:]...))
	}

	// レイアウト確定のためフレームを回す
	for range 10 {
		if err := sm.Update(world); err != nil {
			break
		}
	}
	return &sm
}

// drawStates は駆動済みのステートマシンを1画面へ描画して取り込む。renderMu を保持した状態で呼ぶ。
func drawStates(t *testing.T, world w.World, sm *es.StateMachine[w.World]) *image.NRGBA {
	t.Helper()
	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	for _, state := range sm.GetStates() {
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
