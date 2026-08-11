package menuloop

import (
	"testing"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/vrt"
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
