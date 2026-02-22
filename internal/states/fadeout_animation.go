package states

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
)

// FadeoutAnimationState はフェードアウトアニメーションを行うステート
// フェードアウト完了後に次のステートに遷移する
type FadeoutAnimationState struct {
	es.BaseState[w.World]
	nextStateFunc es.StateFactory[w.World]
	elapsedMs     float64
	fadeMs        float64
	overlay       *ebiten.Image
}

// NewFadeoutAnimationState はフェードアウトアニメーションステートを生成するファクトリを返す
func NewFadeoutAnimationState(nextStateFunc es.StateFactory[w.World]) es.StateFactory[w.World] {
	return func() es.State[w.World] {
		return &FadeoutAnimationState{
			nextStateFunc: nextStateFunc,
			fadeMs:        400.0,
		}
	}
}

func (st FadeoutAnimationState) String() string {
	return "FadeoutAnimation"
}

var _ es.State[w.World] = &FadeoutAnimationState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *FadeoutAnimationState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *FadeoutAnimationState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *FadeoutAnimationState) OnStart(world w.World) error {
	screenW, screenH := world.Resources.GetScreenDimensions()
	st.overlay = ebiten.NewImage(screenW, screenH)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *FadeoutAnimationState) OnStop(_ w.World) error { return nil }

// Update はフェードアウトの進行を管理する
func (st *FadeoutAnimationState) Update(world w.World) (es.Transition[w.World], error) {
	// アニメーション無効時は即座に遷移
	if world.Config.DisableAnimation {
		return es.Transition[w.World]{
			Type:          es.TransReplace,
			NewStateFuncs: []es.StateFactory[w.World]{st.nextStateFunc},
		}, nil
	}

	const deltaMs = 1000.0 / 60.0
	st.elapsedMs += deltaMs

	// フェードアウト完了後に次のステートへ遷移
	if st.elapsedMs >= st.fadeMs {
		return es.Transition[w.World]{
			Type:          es.TransReplace,
			NewStateFuncs: []es.StateFactory[w.World]{st.nextStateFunc},
		}, nil
	}

	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// Draw はフェードアウトエフェクトを描画する
func (st *FadeoutAnimationState) Draw(_ w.World, screen *ebiten.Image) error {
	if st.overlay == nil {
		return nil
	}

	alpha := st.elapsedMs / st.fadeMs
	if alpha > 1.0 {
		alpha = 1.0
	}

	st.overlay.Fill(color.RGBA{0, 0, 0, uint8(alpha * 255)})
	screen.DrawImage(st.overlay, nil)
	return nil
}
