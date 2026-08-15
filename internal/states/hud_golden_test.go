package states_test

// HUD は 3D の世界レイヤの上へ重なる2Dオーバーレイで、3D VRT の命令列には出ない。
// ピクセルが安定した2D描画なので、UIステートと同じピクセル golden で単体固定する。

import (
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
// メッセージ枠の配置とフォントを覆う。世界レイヤは描かず、透明背景へ HUD レイヤだけを描いて
// HUD が塗るピクセルだけを固定する。
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

		key := sysrender.HUDRenderingSystem{}.String()
		updater, uok := world.Updaters[key]
		renderer, rok := world.Renderers[key]
		require.True(t, uok && rok, "HUDRenderingSystem が world に登録されていない")
		// メッセージログは Update で UI を初期化しないと Draw が early-return して描かれない。
		// 他ウィジェットは Draw が毎回データを再抽出するため Update は不要
		require.NoError(t, updater.Update(world))

		return func(screen *ebiten.Image) {
			require.NoError(t, renderer.Draw(world, screen))
		}
	}, consts.GameWidth, consts.GameHeight)
}
