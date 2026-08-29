package menuloop

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// renderKeycaps が ebiten.Image への描画とフォント計測を行うため必要
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestHasEscapeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		table []keybind.Binding
		want  bool
	}{
		{"Escapeキーにラベルがあれば真を返す", []keybind.Binding{{Key: ebiten.KeyEscape, Label: "Back"}}, true},
		{"Escapeキーがラベル空なら偽を返す", []keybind.Binding{{Key: ebiten.KeyEscape, Label: ""}}, false},
		{"Escapeキーが表に無ければ偽を返す", []keybind.Binding{{Key: ebiten.KeyEnter, Label: "Confirm"}}, false},
		{"空の表なら偽を返す", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, hasEscapeLabel(tt.table))
		})
	}
}

func TestRenderKeycaps(t *testing.T) {
	t.Parallel()

	t.Run("トークンが無くても最低1pxの幅を確保する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t, testutil.WithUI())

		img := renderKeycaps(nil, world.Resources.UIResources)

		assert.Equal(t, 1, img.Bounds().Dx())
	})

	t.Run("トークンが増えるほど画像が広がる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t, testutil.WithUI())

		one := renderKeycaps([]string{"A"}, world.Resources.UIResources)
		two := renderKeycaps([]string{"A", "B"}, world.Resources.UIResources)

		assert.Greater(t, two.Bounds().Dx(), one.Bounds().Dx())
	})
}

func TestNewKeyHelpState_渡した表を保持したstateを返す(t *testing.T) {
	t.Parallel()
	table := []keybind.Binding{{Key: ebiten.KeyEnter, Label: "Confirm"}}
	factory := NewKeyHelpState(table)

	state, err := factory()

	require.NoError(t, err)
	st, ok := state.(*KeyHelpState)
	require.True(t, ok, "*KeyHelpStateを返す")
	assert.Equal(t, table, st.table)
}

func TestKeyHelpState_OnStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		table []keybind.Binding
	}{
		{"Escapeの表示行が無い表でも組める", []keybind.Binding{{Key: ebiten.KeyEnter, Action: inputmapper.ActionMenuSelect, Label: "Confirm"}}},
		{"Escapeの表示行がある表でも組める", []keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t, testutil.WithUI())
			st := &KeyHelpState{table: tt.table}

			require.NoError(t, st.OnStart(world))

			assert.NotNil(t, st.body)
		})
	}
}

func TestKeyHelpState_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action inputmapper.ActionID
		ok     bool
		want   es.TransType
	}{
		{"閉じるActionならPopの遷移を返す", inputmapper.ActionCloseMenu, true, es.TransPop},
		{"閉じるAction以外なら遷移しない", "", false, es.TransNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t, testutil.WithUI())
			world.Resources.InputSource = func() (inputmapper.ActionID, bool) {
				return tt.action, tt.ok
			}
			st := &KeyHelpState{table: []keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}}}
			require.NoError(t, st.OnStart(world))

			trans, err := st.Update(world)

			require.NoError(t, err)
			assert.Equal(t, tt.want, trans.Type)
		})
	}
}

func TestKeyHelpState_Draw_組んだ一覧を描く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	st := &KeyHelpState{table: []keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}}}
	require.NoError(t, st.OnStart(world))

	require.NoError(t, st.Draw(world, ebiten.NewImage(10, 10)))
}
