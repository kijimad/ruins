package states_test

// HUD は 3D の世界レイヤの上へ重なる2Dオーバーレイで、3D VRT の命令列には出ない。
// ピクセルが安定した2D描画なので、UIステートと同じピクセル golden で単体固定する。
// world.Renderers に登録された enabled な実体を引き、実ゲームと同じ Update→Draw で描く。

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	sysrender "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/stretchr/testify/require"
)

// TestGolden_HUD はダンジョン上の標準 HUD を固定する。HP・武器スロット・通貨・ミニマップ・
// メッセージ枠の配置とフォントを覆う。世界レイヤは描かず HUD レイヤだけを黒背景へ描く。
func TestGolden_HUD(t *testing.T) {
	t.Parallel()

	world := vrt.InitVRTWorld(t)

	vrt.AssertScreenGolden(t, func() func(screen *ebiten.Image) {
		st := &gs.DungeonState{
			Depth:          1,
			DefinitionName: dungeon.DungeonDebug.Name(),
			BuilderType:    mapplanner.PlannerTypeSmallRoom,
		}
		// OnStart がプレイヤーを湧かせ、HUD システムを world.Renderers へ登録する
		require.NoError(t, st.OnStart(world))

		hud, ok := world.Renderers[sysrender.HUDRenderingSystem{}.String()].(*sysrender.HUDRenderingSystem)
		require.True(t, ok, "HUDRenderingSystem が world.Renderers に登録されていない")
		// 実ゲームと同じく Update で各ウィジェットのデータを反映してから描く
		require.NoError(t, hud.Update(world))

		return func(screen *ebiten.Image) {
			screen.Fill(color.Black)
			require.NoError(t, hud.Draw(world, screen))
		}
	}, consts.GameWidth, consts.GameHeight)
}
