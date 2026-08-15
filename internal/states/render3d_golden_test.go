package states_test

import (
	"encoding/json"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/states"
	sysrender "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// render3DSceneSeed は撮影シーンの生成シード。キャラとポータルが重ならない見栄えの良い配置を選ぶ。
const render3DSceneSeed = 3

// render3DScene は3D VRTで固定するシーンを組む。TestGolden_Overworld と同じ昼間のオーバーワールドを
// 流用する。ダンジョンは暗闇が多く3Dの見た目が沈むため、明るく開けた本流ステージを撮影対象にする。
// 固定シードで決定的に生成する。
func render3DScene(t *testing.T) func(w.World) []es.State[w.World] {
	t.Helper()
	return func(_ w.World) []es.State[w.World] {
		s, err := gs.NewOverworldState(
			mapplanner.PlannerTypeOverworldField,
			dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1),
			&overworld.NewGameParams{RunSeed: render3DSceneSeed},
		)()
		require.NoError(t, err)
		return []es.State[w.World]{s}
	}
}

// draw3DList は既定カメラの Render3DSystem で命令列を取り出す。DrawList は whitePixel の
// 生成やアトラス読みで ebiten グローバルに触れるため WithUILock 内で呼ぶ。
func draw3DList(world w.World) []sysrender.R3DrawCommand {
	sys := sysrender.NewRender3DSystem()
	var cmds []sysrender.R3DrawCommand
	vrt.WithUILock(func() {
		cmds = sys.DrawList(world, consts.GameWidth, consts.GameHeight)
	})
	return cmds
}

// TestGolden_Render3DSnapshot は3Dレンダラの命令列をJSONで固定する。ピクセルが不安定でも、
// ソート後・投影後のクアッド列を真実源にすれば退行を決定的に検知できる。座標や色は整数へ
// 丸めて float ノイズを排除する。命令列が変わったら差分を読み、意図した変更か確認する。
func TestGolden_Render3DSnapshot(t *testing.T) {
	t.Parallel()

	world := vrt.BuildWorld(t, render3DScene(t))
	cmds := draw3DList(world)

	data, err := json.MarshalIndent(cmds, "", "  ")
	require.NoError(t, err)

	g := goldie.New(t, goldie.WithNameSuffix(".json"))
	g.Assert(t, t.Name(), data)
}

// TestRender3DImages は命令列JSONが変わったときだけ3Dシーンのスクショを保存する。
// ピクセル比較はせず、目視確認用の参照画像として置く。3D描画はピクセルが不安定なので
// pass/fail は TestGolden_Render3DSnapshot のJSONに委ね、画像は成果物として更新する。
func TestRender3DImages(t *testing.T) {
	t.Parallel()

	world := vrt.BuildWorld(t, render3DScene(t))
	cmds := draw3DList(world)
	currentJSON, err := json.MarshalIndent(cmds, "", "  ")
	require.NoError(t, err)

	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	imgPath := g.GoldenFileName(t, t.Name())
	// pass/fail の真実源である命令列JSONと突き合わせる。JSONが変わった時だけ画像を作り直す
	jsonPath := filepath.Join("testdata", "TestGolden_Render3DSnapshot.json")
	if !imgNeedsUpdate(imgPath, jsonPath, currentJSON) {
		return
	}

	pngData := vrt.RenderWorldPNG(t, render3DScene(t), func(world w.World, screen *ebiten.Image) {
		// ゲームと同じく黒背景の上に3Dシーンを描く
		screen.Fill(color.Black)
		sys := sysrender.NewRender3DSystem()
		_ = sys.Draw(world, screen)
	})
	require.NoError(t, g.Update(t, t.Name(), pngData))
	t.Logf("画像を更新: %s", imgPath)
}
