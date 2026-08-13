package menuloop

import (
	"errors"
	"image"
	"testing"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dirtyTestModel は View の呼び出し回数を数えるだけの最小 Model。dirty ゲートの検証に使う。
// props は int で、テストが値を差し替えて再構築の有無を観測する
type dirtyTestModel struct {
	props     int
	viewCount int
}

func (m *dirtyTestModel) ConsumeTransition() es.Transition[w.World] { return es.Transition[w.World]{} }

func (m *dirtyTestModel) DoAction(_ w.World, _ inputmapper.ActionID) (es.Transition[w.World], error) {
	return es.Transition[w.World]{}, nil
}

func (m *dirtyTestModel) Fetch(_ w.World) int { return m.props }

func (m *dirtyTestModel) Menu(_ int) MenuConfig { return MenuConfig{Key: "test", TabCount: 0} }

func (m *dirtyTestModel) View(_ w.World, _ int, _ Selection, _ resources.UIResources) *ebitenui.UI {
	m.viewCount++
	return &ebitenui.UI{Container: widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))}
}

// flexModel は DoAction・ExtraInput の挙動を差し替えられる Model[int] のテストダブル。
// Update・readAction の分岐を個別に固定するのに使う
type flexModel struct {
	props         int
	menu          MenuConfig
	doAction      func(w.World, inputmapper.ActionID) (es.Transition[w.World], error)
	extraInput    func() (inputmapper.ActionID, bool)
	doActionCalls int
}

func (m *flexModel) ConsumeTransition() es.Transition[w.World] { return es.Transition[w.World]{} }

func (m *flexModel) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	m.doActionCalls++
	if m.doAction != nil {
		return m.doAction(world, action)
	}
	return es.Transition[w.World]{}, nil
}

func (m *flexModel) Fetch(_ w.World) int { return m.props }

func (m *flexModel) Menu(_ int) MenuConfig { return m.menu }

func (m *flexModel) View(_ w.World, _ int, _ Selection, _ resources.UIResources) *ebitenui.UI {
	return &ebitenui.UI{Container: widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))}
}

// ExtraInput は menuloop.ExtraInput を満たす。extraInput が未設定なら常にフォールバックさせる
func (m *flexModel) ExtraInput() (inputmapper.ActionID, bool) {
	if m.extraInput != nil {
		return m.extraInput()
	}
	return "", false
}

var _ ExtraInput = (*flexModel)(nil)

// testOverlay は overlay.Layer のテストダブル。Active・HandleInput の呼び出しを観測する
type testOverlay struct {
	active           bool
	handleInputErr   error
	handleInputCalls int
	window           *widget.Window
}

func (o *testOverlay) Active() bool { return o.active }

func (o *testOverlay) HandleInput(_ w.World) error {
	o.handleInputCalls++
	return o.handleInputErr
}

func (o *testOverlay) Window(_ w.World, _ image.Rectangle) *widget.Window { return o.window }

var _ overlay.Layer = (*testOverlay)(nil)

// TestScreen_Props_更新後の値を返す は Props が mount 経由の現在値を読めることを固定する
func TestScreen_Props_更新後の値を返す(t *testing.T) {
	t.Parallel()
	model := &flexModel{props: 5, menu: MenuConfig{Key: "props"}}
	screen := NewScreen[int](model)
	world := w.World{Resources: &resources.Resources{}}

	vrt.WithUILock(func() {
		_, err := screen.Update(world)
		require.NoError(t, err)
	})

	assert.Equal(t, 5, screen.Props())
}

// TestScreen_SetTab は範囲内のタブ移動と範囲外の無視を固定する
func TestScreen_SetTab(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tab     int
		wantTab int
	}{
		{"範囲内のタブに移動する", 1, 1},
		{"負のタブ番号は無視する", -1, 0},
		{"範囲外の大きいタブ番号は無視する", 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := &flexModel{menu: MenuConfig{Key: "settab", TabCount: 3, ItemCounts: []int{1, 1, 1}}}
			screen := NewScreen[int](model)
			world := w.World{Resources: &resources.Resources{}}

			// UseTabMenu で初期状態を登録してから SetTab で動かす
			vrt.WithUILock(func() {
				_, err := screen.Update(world)
				require.NoError(t, err)
			})

			screen.SetTab(tt.tab)

			got, ok := hooks.GetState[hooks.TabMenuState](screen.mount, "settab")
			require.True(t, ok)
			assert.Equal(t, tt.wantTab, got.TabIndex)
		})
	}
}

// TestScreen_SetTab_TabCountが0なら何もしない は一覧を持たない画面で panic しないことを固定する
func TestScreen_SetTab_TabCountが0なら何もしない(t *testing.T) {
	t.Parallel()
	model := &flexModel{menu: MenuConfig{Key: "settabzero", TabCount: 0}}
	screen := NewScreen[int](model)

	assert.NotPanics(t, func() {
		screen.SetTab(1)
	})
}

// TestScreen_Selection_初回はゼロ値を返す は Update 前の Selection がゼロ値であることを固定する
func TestScreen_Selection_初回はゼロ値を返す(t *testing.T) {
	t.Parallel()
	model := &flexModel{menu: MenuConfig{Key: "sel", TabCount: 2, ItemCounts: []int{1, 1}}}
	screen := NewScreen[int](model)

	assert.Equal(t, Selection{}, screen.Selection())
}

// TestScreen_Draw は widget 未生成時と生成後の両方で panic しないことを固定する
func TestScreen_Draw(t *testing.T) {
	t.Parallel()
	t.Run("widget未生成ならpanicしない", func(t *testing.T) {
		t.Parallel()
		model := &flexModel{menu: MenuConfig{Key: "draw1"}}
		screen := NewScreen[int](model)

		assert.NotPanics(t, func() {
			screen.Draw(ebiten.NewImage(1, 1))
		})
	})

	t.Run("Update後は保持しているUIを描画する", func(t *testing.T) {
		t.Parallel()
		model := &flexModel{menu: MenuConfig{Key: "draw2"}}
		screen := NewScreen[int](model)
		world := w.World{Resources: &resources.Resources{}}

		vrt.WithUILock(func() {
			_, err := screen.Update(world)
			require.NoError(t, err)
		})

		assert.NotPanics(t, func() {
			screen.Draw(ebiten.NewImage(10, 10))
		})
	})
}

// TestScreen_activeOverlay は登録順で最初の Active な overlay を返すことを固定する
func TestScreen_activeOverlay(t *testing.T) {
	t.Parallel()

	t.Run("Activeなoverlayが無ければnilを返す", func(t *testing.T) {
		t.Parallel()
		model := &flexModel{menu: MenuConfig{Key: "ov1"}}
		ov := &testOverlay{active: false}
		screen := NewScreen[int](model, ov)

		assert.Nil(t, screen.activeOverlay())
	})

	t.Run("登録順で最初のActiveなoverlayを返す", func(t *testing.T) {
		t.Parallel()
		model := &flexModel{menu: MenuConfig{Key: "ov2"}}
		ov1 := &testOverlay{active: false}
		ov2 := &testOverlay{active: true}
		screen := NewScreen[int](model, ov1, ov2)

		assert.Same(t, ov2, screen.activeOverlay())
	})
}

// TestScreen_readAction は ExtraInput が真ならそちらを優先し、
// 偽なら HandleMenuInput にフォールバックすることを固定する
func TestScreen_readAction(t *testing.T) {
	t.Parallel()

	t.Run("ExtraInputが真なら優先する", func(t *testing.T) {
		t.Parallel()
		model := &flexModel{
			menu:       MenuConfig{Key: "extra1"},
			extraInput: func() (inputmapper.ActionID, bool) { return inputmapper.ActionMenuSelect, true },
		}
		screen := NewScreen[int](model)

		action, ok := screen.readAction()

		assert.True(t, ok)
		assert.Equal(t, inputmapper.ActionMenuSelect, action)
	})

	t.Run("ExtraInputが偽ならHandleMenuInputにフォールバックする", func(t *testing.T) {
		t.Parallel()
		model := &flexModel{menu: MenuConfig{Key: "extra2"}}
		screen := NewScreen[int](model)

		action, ok := screen.readAction()

		assert.False(t, ok)
		assert.Equal(t, inputmapper.ActionID(""), action)
	})
}

// TestScreen_Update_overlay は Active な overlay が入力を専有し、DoAction を呼ばないことを固定する
func TestScreen_Update_overlay(t *testing.T) {
	t.Parallel()

	t.Run("ActiveならHandleInputを呼びDoActionを呼ばない", func(t *testing.T) {
		t.Parallel()
		model := &flexModel{menu: MenuConfig{Key: "ovupdate"}}
		ov := &testOverlay{active: true}
		screen := NewScreen[int](model, ov)
		world := w.World{Resources: &resources.Resources{}}

		vrt.WithUILock(func() {
			_, err := screen.Update(world)
			require.NoError(t, err)
		})

		assert.Equal(t, 1, ov.handleInputCalls)
		assert.Equal(t, 0, model.doActionCalls)
	})

	t.Run("HandleInputがエラーを返せばUpdateもエラーを返す", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("overlay input failed")
		model := &flexModel{menu: MenuConfig{Key: "ovupdateerr"}}
		ov := &testOverlay{active: true, handleInputErr: wantErr}
		screen := NewScreen[int](model, ov)
		world := w.World{Resources: &resources.Resources{}}

		var err error
		vrt.WithUILock(func() {
			_, err = screen.Update(world)
		})

		assert.ErrorIs(t, err, wantErr)
	})
}

// TestScreen_Update_DoAction は DoAction の遷移・エラーがそのまま返ることを固定する
func TestScreen_Update_DoAction(t *testing.T) {
	t.Parallel()

	t.Run("遷移を返したらそのまま返す", func(t *testing.T) {
		t.Parallel()
		wantTrans := es.Transition[w.World]{Type: es.TransPop}
		model := &flexModel{
			menu:       MenuConfig{Key: "trans"},
			extraInput: func() (inputmapper.ActionID, bool) { return inputmapper.ActionMenuSelect, true },
			doAction: func(_ w.World, _ inputmapper.ActionID) (es.Transition[w.World], error) {
				return wantTrans, nil
			},
		}
		screen := NewScreen[int](model)
		world := w.World{Resources: &resources.Resources{}}

		var got es.Transition[w.World]
		var err error
		vrt.WithUILock(func() {
			got, err = screen.Update(world)
		})

		require.NoError(t, err)
		assert.Equal(t, wantTrans, got)
		assert.Equal(t, 1, model.doActionCalls)
	})

	t.Run("エラーを返せばUpdateもエラーを返す", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("do action failed")
		model := &flexModel{
			menu:       MenuConfig{Key: "transerr"},
			extraInput: func() (inputmapper.ActionID, bool) { return inputmapper.ActionMenuSelect, true },
			doAction: func(_ w.World, _ inputmapper.ActionID) (es.Transition[w.World], error) {
				return es.Transition[w.World]{}, wantErr
			},
		}
		screen := NewScreen[int](model)
		world := w.World{Resources: &resources.Resources{}}

		var err error
		vrt.WithUILock(func() {
			_, err = screen.Update(world)
		})

		assert.ErrorIs(t, err, wantErr)
	})
}

// TestScreen_dirtyGateは変化時だけViewを組み直す は retained 化の核を固定する。
// props もカーソルも overlay も動かないフレームでは View を再構築せず、変化したフレームだけ組み直す
func TestScreen_dirtyGateは変化時だけViewを組み直す(t *testing.T) {
	t.Parallel()
	model := &dirtyTestModel{props: 1}
	screen := NewScreen[int](model)
	world := w.World{Resources: &resources.Resources{}}

	// widget 生成は ebitenui のグローバル状態に触れるのでロックで直列化する
	vrt.WithUILock(func() {
		_, err := screen.Update(world)
		require.NoError(t, err)
		assert.Equal(t, 1, model.viewCount, "初回は必ず組む")

		_, err = screen.Update(world)
		require.NoError(t, err)
		assert.Equal(t, 1, model.viewCount, "props もカーソルも不変なら再構築しない")

		model.props = 2
		_, err = screen.Update(world)
		require.NoError(t, err)
		assert.Equal(t, 2, model.viewCount, "props が変われば再構築する")

		_, err = screen.Update(world)
		require.NoError(t, err)
		assert.Equal(t, 2, model.viewCount, "再び不変なら据え置く")
	})
}
