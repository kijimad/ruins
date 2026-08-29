package menuloop

import (
	"os"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// renderKeycaps が ebiten.Image への描画とフォント計測を行うため必要
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

// newKeyHelpWorld はキー一覧ヘルプのテスト用に実UIリソースを積んだworldを返す
func newKeyHelpWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)
	return world
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

func TestKeyHelpRow(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)

	tests := []struct {
		name  string
		entry keybind.HintEntry
	}{
		{"単一トークンの行を組む", keybind.HintEntry{Keys: "Esc", Label: "Back", Tokens: []string{"Esc"}}},
		{"複数トークンの行を組む", keybind.HintEntry{Keys: "↑↓", Label: "Select", Tokens: []string{"↑", "↓"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var row *widget.Container
			vrt.WithUILock(func() {
				row = keyHelpRow(tt.entry, res)
			})

			children := row.Children()
			require.Len(t, children, 2, "キーキャップ画像とラベルの2要素を持つ")

			graphic, ok := children[0].(*widget.Graphic)
			require.True(t, ok, "先頭はキーキャップ画像")
			assert.Positive(t, graphic.Image.Bounds().Dx(), "画像に幅がある")

			label, ok := children[1].(*widget.Text)
			require.True(t, ok, "末尾はラベルテキスト")
			assert.Equal(t, tt.entry.Label, label.Label)
		})
	}
}

func TestRenderKeycaps(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)

	t.Run("トークンが無くても最低1pxの幅を確保する", func(t *testing.T) {
		t.Parallel()
		var img *ebiten.Image
		vrt.WithUILock(func() {
			img = renderKeycaps(nil, res)
		})
		assert.Equal(t, 1, img.Bounds().Dx())
	})

	t.Run("トークンが増えるほど画像が広がる", func(t *testing.T) {
		t.Parallel()
		var one, two *ebiten.Image
		vrt.WithUILock(func() {
			one = renderKeycaps([]string{"A"}, res)
			two = renderKeycaps([]string{"A", "B"}, res)
		})
		assert.Greater(t, two.Bounds().Dx(), one.Bounds().Dx())
	})
}

func TestNewKeyHelpState(t *testing.T) {
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
			world := newKeyHelpWorld(t)
			st := &KeyHelpState{table: tt.table}

			var err error
			vrt.WithUILock(func() {
				err = st.OnStart(world)
			})

			require.NoError(t, err)
			assert.NotNil(t, st.widget)
		})
	}
}

func TestKeyHelpState_Update(t *testing.T) {
	t.Parallel()

	t.Run("閉じるActionならPopの遷移を返す", func(t *testing.T) {
		t.Parallel()
		world := newKeyHelpWorld(t)
		world.Resources.InputSource = func() (inputmapper.ActionID, bool) {
			return inputmapper.ActionCloseMenu, true
		}
		st := &KeyHelpState{table: []keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}}}

		var trans es.Transition[w.World]
		var err error
		vrt.WithUILock(func() {
			require.NoError(t, st.OnStart(world))
			trans, err = st.Update(world)
		})

		require.NoError(t, err)
		assert.Equal(t, es.TransPop, trans.Type)
	})

	t.Run("閉じるAction以外なら遷移せずUIを更新する", func(t *testing.T) {
		t.Parallel()
		world := newKeyHelpWorld(t)
		world.Resources.InputSource = func() (inputmapper.ActionID, bool) {
			return "", false
		}
		st := &KeyHelpState{table: []keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}}}

		var trans es.Transition[w.World]
		var err error
		vrt.WithUILock(func() {
			require.NoError(t, st.OnStart(world))
			trans, err = st.Update(world)
		})

		require.NoError(t, err)
		assert.Equal(t, es.TransNone, trans.Type)
	})
}

func TestKeyHelpState_Draw(t *testing.T) {
	t.Parallel()
	world := newKeyHelpWorld(t)
	st := &KeyHelpState{table: []keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}}}

	vrt.WithUILock(func() {
		require.NoError(t, st.OnStart(world))

		assert.NotPanics(t, func() {
			require.NoError(t, st.Draw(world, ebiten.NewImage(10, 10)))
		})
	})
}
