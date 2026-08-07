package states_test

// 詳細モーダルは showDetail が真のときだけ描画される。未公開フィールドには触れず、
// 公開メソッド Update/DoAction/Draw でモーダルを開いた状態を作って固定する。
// vrt は maingame 経由で states を import するため、この golden は外部テストに置く。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/require"
)

// TestGolden_ItemActionDetail は x で開く詳細モーダルの描画を固定する。
// 個数とタイトルバーが無く、性能・性質と説明が並ぶことを覆う。
func TestGolden_ItemActionDetail(t *testing.T) {
	t.Parallel()

	world := vrt.InitVRTWorld(t)
	_, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3)
	require.NoError(t, err)

	vrt.AssertScreenGolden(t, func() func(screen *ebiten.Image) {
		st := &gs.ItemActionState{}
		require.NoError(t, st.OnStart(world))
		// props と widget を構築する
		_, err := st.Update(world)
		require.NoError(t, err)
		// 調べるタブ先頭アイテムの詳細モーダルを開く
		_, err = st.DoAction(world, inputmapper.ActionOpenItemDetail)
		require.NoError(t, err)
		// 開いた状態で UI を作り直す
		_, err = st.Update(world)
		require.NoError(t, err)

		return func(screen *ebiten.Image) {
			require.NoError(t, st.Draw(world, screen))
		}
	}, consts.GameWidth, consts.GameHeight)
}
