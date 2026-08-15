package states_test

import (
	"encoding/json"
	"image/color"
	"path/filepath"
	"slices"
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
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// render3DSceneSeed はオーバーワールドの RunSeed。キャラとポータルが重ならない見栄えの良い配置を選ぶ。
const render3DSceneSeed = 3

// r3Scene は3D VRTで固定するワールドシーン。ワールドを映すVRTはすべて3D命令列で撮る。
// prep は描画直前に world を追加調整する。nil可。前線を可視域へ入れるなどに使う。
type r3Scene struct {
	name  string
	build func(w.World) []es.State[w.World]
	prep  func(w.World)
}

// overworldBuild は昼間のオーバーワールドを固定シードで組む。通常と霜シーンで共有する。
func overworldBuild(t *testing.T) func(w.World) []es.State[w.World] {
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

// r3Scenes は撮影シーン一覧を返す。昼間オーバーワールド、壁を含むダンジョン、寒波前線の氷を覆う。
func r3Scenes(t *testing.T) []r3Scene {
	t.Helper()
	return []r3Scene{
		{
			name:  "Overworld",
			build: overworldBuild(t),
		},
		{
			name: "Dungeon",
			build: func(_ w.World) []es.State[w.World] {
				return []es.State[w.World]{&gs.DungeonState{
					Depth:          1,
					DefinitionName: dungeon.DungeonDebug.Name(),
					BuilderType:    mapplanner.PlannerTypeSmallRoom,
				}}
			},
			// プレイヤーは上り階段の上に湧くため、そのままだと階段のポータルと重なる。
			// 通行可能な隣接床へ1マスずらして重なりを避ける
			prep: func(world w.World) {
				pe, err := query.GetPlayerEntity(world)
				if err != nil {
					return
				}
				g := world.Components.GridElement.Get(pe)
				if p, ok := passableNeighbor(world, g.Coord); ok {
					g.Coord = p
				}
			},
		},
		{
			name:  "OverworldFrost",
			build: overworldBuild(t),
			// 前線をプレイヤーより十分東へ置き、可視域を凍結ゾーンにする。前線位置は本来ターンから
			// 導出するが、VRTは霜の描画経路を撮るのが目的なので位置を直接指定して確実に凍らせる
			prep: func(world w.World) {
				sb := query.GetSeamlessBand(world)
				if pe, err := query.GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(pe) {
					px := world.Components.GridElement.Get(pe).X
					sb.Front.EastAbsX = sb.LocalToAbsX(px) + 1000
				}
			},
		},
	}
}

// passableNeighbor は通行可能で床タイルのある隣接マスを返す。右左下上の順で最初に見つかったもの。
// void へ動かさないよう床タイルの存在も確かめる。
func passableNeighbor(world w.World, from consts.Coord[consts.Tile]) (consts.Coord[consts.Tile], bool) {
	si := query.GetSpatialIndex(world)
	for _, d := range []consts.Coord[consts.Tile]{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		p := consts.Coord[consts.Tile]{X: from.X + d.X, Y: from.Y + d.Y}
		if si.IsBlockPass(p) {
			continue
		}
		if slices.ContainsFunc(query.GetEntitiesAt(world, p.X, p.Y), world.Components.Tile.Has) {
			return p, true
		}
	}
	return from, false
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
			if sc.prep != nil {
				sc.prep(world)
			}
			cmds := draw3DList(world)

			data, err := json.MarshalIndent(cmds, "", "  ")
			require.NoError(t, err)

			g := goldie.New(t, goldie.WithNameSuffix(".json"))
			g.Assert(t, "TestGolden_3D_"+sc.name, data)
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
			if sc.prep != nil {
				sc.prep(world)
			}
			cmds := draw3DList(world)
			currentJSON, err := json.MarshalIndent(cmds, "", "  ")
			require.NoError(t, err)

			name := "TestGolden_3D_" + sc.name
			g := goldie.New(t, goldie.WithNameSuffix(".png"))
			imgPath := g.GoldenFileName(t, name)
			// pass/fail の真実源である命令列JSONと突き合わせる。JSONが変わった時だけ画像を作り直す
			jsonPath := filepath.Join("testdata", name+".json")
			if !imgNeedsUpdate(imgPath, jsonPath, currentJSON) {
				return
			}

			pngData := vrt.RenderPNG(t, sc.build, func(world w.World, screen *ebiten.Image) {
				if sc.prep != nil {
					sc.prep(world)
				}
				// ゲームと同じく黒背景の上に3Dシーンを描く
				screen.Fill(color.Black)
				sys := sysrender.NewRender3DSystem()
				_ = sys.Draw(world, screen)
			})
			require.NoError(t, g.Update(t, name, pngData))
			t.Logf("画像を更新: %s", imgPath)
		})
	}
}
