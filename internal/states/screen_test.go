package states_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestChoiceMenu は2択の選択メニューを返す。先頭を選ぶと ran を true にして Pop する
func newTestChoiceMenu(ran *bool) *gs.ChoiceMenuState {
	return gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) {
		return "タイトル", []gs.Choice{
			{Label: "実行", Run: func(_ w.World) (es.Transition[w.World], error) {
				*ran = true
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}},
			{Label: "閉じる"},
		}
	})
}

func TestScreen_ChoiceMenu_選択で先頭のRunが走りその遷移を返す(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	var ran bool
	st := newTestChoiceMenu(&ran)

	require.NoError(t, st.OnStart(world))
	// Update で props と選択位置が確定し、先頭にカーソルが載る
	var err error
	_, err = st.Update(world)
	require.NoError(t, err)

	tr, err := st.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.True(t, ran, "選択した先頭の Run が実行される")
	assert.Equal(t, es.TransPop, tr.Type, "Run が返した遷移がそのまま返る")
}

func TestScreen_ChoiceMenu_キャンセルでPop(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	var ran bool
	st := newTestChoiceMenu(&ran)

	require.NoError(t, st.OnStart(world))
	tr, err := st.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, tr.Type)
	assert.False(t, ran, "キャンセルでは Run は走らない")
}

func TestScreen_ChoiceMenu_UpdateとDrawが通る(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	var ran bool
	st := newTestChoiceMenu(&ran)

	require.NoError(t, st.OnStart(world))
	// Update と Draw が描画までパニックせず通ることを確認する
	var err error
	if _, err = st.Update(world); err != nil {
		return
	}
	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	err = st.Draw(world, screen)
	require.NoError(t, err)
}

func TestScreen_ChoiceMenu_見出し行はカーソルが飛ばされる(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	var ran bool
	st := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) {
		return "", []gs.Choice{
			{Label: "見出し", Header: true},
			{Label: "実行", Run: func(_ w.World) (es.Transition[w.World], error) {
				ran = true
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}},
		}
	})
	require.NoError(t, st.OnStart(world))
	var err error
	_, err = st.Update(world)
	require.NoError(t, err)

	// 先頭は見出し行なのでカーソルは実行行へ飛ばされ、選択で Run が走る
	tr, err := st.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.True(t, ran, "見出しを飛ばし実行行が選ばれる")
	assert.Equal(t, es.TransPop, tr.Type)
}
