package states_test

import (
	"encoding/json"
	"image/color"
	"math/rand/v2"
	"path/filepath"
	"strings"
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

// 撮影シーンの生成シード。いずれもキャラとポータルが重ならない見栄えの良い配置を選ぶ。
const (
	// render3DSceneSeed はオーバーワールドの RunSeed
	render3DSceneSeed = 3
	// render3DDungeonSeed はダンジョン層生成の RNG シード。層生成は world.Config.RNG から
	// シードを引くので、VRT共有の 12345 では重なる。この撮影用に RNG を差し替える
	render3DDungeonSeed = 7
)

// r3Scene は3D VRTで固定するワールドシーン。ワールドを映すVRTはすべて3D命令列で撮る。
type r3Scene struct {
	name  string
	build func(w.World) []es.State[w.World]
}

// r3Scenes は撮影シーン一覧を返す。明るい昼間のオーバーワールドと、壁を含むダンジョンを覆う。
func r3Scenes(t *testing.T) []r3Scene {
	t.Helper()
	return []r3Scene{
		{
			name: "Overworld",
			build: func(_ w.World) []es.State[w.World] {
				s, err := gs.NewOverworldState(
					mapplanner.PlannerTypeOverworldField,
					dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1),
					&overworld.NewGameParams{RunSeed: render3DSceneSeed},
				)()
				require.NoError(t, err)
				return []es.State[w.World]{s}
			},
		},
		{
			name: "Dungeon",
			build: func(world w.World) []es.State[w.World] {
				// 層生成のRNGを撮影用に差し替える。VRT共有シードだとキャラとポータルが重なる。
				// テストごとに固有の world なので他テストへ波及しない
				world.Config.RNG = rand.New(rand.NewPCG(render3DDungeonSeed, 0))
				return []es.State[w.World]{&gs.DungeonState{
					Depth:          1,
					DefinitionName: dungeon.DungeonDebug.Name(),
					BuilderType:    mapplanner.PlannerTypeSmallRoom,
				}}
			},
		},
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
	for _, sc := range r3Scenes(t) {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			world := vrt.BuildWorld(t, sc.build)
			cmds := draw3DList(world)

			data, err := json.MarshalIndent(cmds, "", "  ")
			require.NoError(t, err)

			g := goldie.New(t, goldie.WithNameSuffix(".json"))
			g.Assert(t, t.Name(), data)
		})
	}
}

// TestRender3DImages は命令列JSONが変わったときだけ3Dシーンのスクショを保存する。
// ピクセル比較はせず、目視確認用の参照画像として置く。3D描画はピクセルが不安定なので
// pass/fail は TestGolden_Render3DSnapshot のJSONに委ね、画像は成果物として更新する。
func TestRender3DImages(t *testing.T) {
	t.Parallel()
	for _, sc := range r3Scenes(t) {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			world := vrt.BuildWorld(t, sc.build)
			cmds := draw3DList(world)
			currentJSON, err := json.MarshalIndent(cmds, "", "  ")
			require.NoError(t, err)

			g := goldie.New(t, goldie.WithNameSuffix(".png"))
			imgPath := g.GoldenFileName(t, t.Name())
			// pass/fail の真実源である命令列JSONと突き合わせる。JSONが変わった時だけ画像を作り直す
			subName := strings.TrimPrefix(t.Name(), "TestRender3DImages/")
			jsonPath := filepath.Join("testdata", "TestGolden_Render3DSnapshot", subName+".json")
			if !imgNeedsUpdate(imgPath, jsonPath, currentJSON) {
				return
			}

			pngData := vrt.RenderWorldPNG(t, sc.build, func(world w.World, screen *ebiten.Image) {
				// ゲームと同じく黒背景の上に3Dシーンを描く
				screen.Fill(color.Black)
				sys := sysrender.NewRender3DSystem()
				_ = sys.Draw(world, screen)
			})
			require.NoError(t, g.Update(t, t.Name(), pngData))
			t.Logf("画像を更新: %s", imgPath)
		})
	}
}
