package states

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestItemActionShortcut_文字キーが動詞Actionへ変換される は、メニューの合成済み束縛表で
// 各動詞の文字キーが対応する動詞 Action へ変換されることを固定する。Screen が組む表と同じ
// MustMerge で合成し、キー読みの Convert をモックキーボードで通す。
func TestItemActionShortcut_文字キーが動詞Actionへ変換される(t *testing.T) {
	t.Parallel()

	table := keybind.MustMerge(itemActionBindings, keybind.MenuCommon)
	// ケースは verbList から導く。動詞を1つ足したら自動でここも網羅する。ハードコードだと
	// 追加された動詞、例えば調べる X+Shift、の取りこぼしに気づけない
	for _, v := range verbList {
		t.Run(string(v.ID), func(t *testing.T) {
			t.Parallel()
			mock := input.NewMockKeyboardInput()
			mock.SetKeyJustPressed(v.Key, true)
			if v.Shift {
				mock.SetKeyPressed(ebiten.KeyShift, true)
			}
			got, ok := keybind.Convert(mock, table)
			require.True(t, ok, "%s のキーが Action へ変換される", v.ID)
			assert.Equal(t, v.Action, got)
		})
	}
}

// TestItemActionShortcut_動詞Actionでタブへジャンプする は、動詞メニューを本番と同じ
// Screen.Update ループで駆動し、動詞 Action が対応タブへカーソルを移すことを固定する。
// 置くタブ選択中に食べるの Action を流すと食べるタブへ飛ぶ、というユーザー操作を再現する。
func TestItemActionShortcut_動詞Actionでタブへジャンプする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "healing_potion", 3)
	require.NoError(t, err)

	st := &ItemActionState{initialVerb: verbPlace}
	require.NoError(t, st.OnStart(world))

	// 初期タブを確定させる1フレーム
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return "", false }
	_, err = st.screen.Update(world)
	require.NoError(t, err)
	require.Equal(t, verbTabIndex(verbPlace), st.screen.Selection().TabIndex, "置くタブから始まる")

	// 食べるの Action を1回流す
	fired := false
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) {
		if fired {
			return "", false
		}
		fired = true
		return inputmapper.ActionVerbConsume, true
	}
	_, err = st.screen.Update(world)
	require.NoError(t, err)
	assert.Equal(t, verbTabIndex(verbConsume), st.screen.Selection().TabIndex, "食べるタブへ飛ぶ")
}
