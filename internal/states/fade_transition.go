package states

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
)

// FadeTransitionState は暗転→処理→明転のトランジション演出。
// 同一ステート内のレベル入れ替えなど、画面を切らずに世界を差し替える用途に使う。
// TransPush で下のステートに重ね、暗転しきった瞬間に onBlack を1回実行し、
// その後明転して自身を pop する。下のステートは暗転の裏で差し替わり、明転で現れる。
type FadeTransitionState struct {
	es.BaseState[w.World]
	// onBlack は暗転しきった瞬間に1回だけ実行する。ここで世界を差し替える
	onBlack   func(w.World) error
	elapsedMs float64
	fadeMs    float64
	fired     bool
	overlay   *ebiten.Image
}

// NewFadeTransitionState は暗転→onBlack→明転を行うステートのファクトリを返す
func NewFadeTransitionState(onBlack func(w.World) error) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &FadeTransitionState{onBlack: onBlack, fadeMs: 250.0}, nil
	}
}

func (st FadeTransitionState) String() string { return "FadeTransition" }

var _ es.State[w.World] = &FadeTransitionState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *FadeTransitionState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *FadeTransitionState) OnResume(_ w.World) error { return nil }

// OnStop はステートが停止される際に呼ばれる
func (st *FadeTransitionState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *FadeTransitionState) OnStart(world w.World) error {
	screenW, screenH := world.Resources.GetScreenDimensions()
	st.overlay = ebiten.NewImage(screenW, screenH)
	return nil
}

// fireOnce は onBlack を高々1回実行する
func (st *FadeTransitionState) fireOnce(world w.World) error {
	if st.fired {
		return nil
	}
	st.fired = true
	return st.onBlack(world)
}

// Update は暗転の進行を管理し、暗転点で onBlack を実行、明転後に自身を pop する
func (st *FadeTransitionState) Update(world w.World) (es.Transition[w.World], error) {
	// アニメ無効時は onBlack だけ実行して即 pop する
	if world.Config.DisableAnimation {
		if err := st.fireOnce(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	const deltaMs = 1000.0 / 60.0
	st.elapsedMs += deltaMs

	// 暗転しきった瞬間に世界を差し替える
	if st.elapsedMs >= st.fadeMs {
		if err := st.fireOnce(world); err != nil {
			return es.Transition[w.World]{}, err
		}
	}

	// 明転しきったら自身を pop して下のステートへ戻す
	if st.elapsedMs >= st.fadeMs*2 {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// Draw は暗転オーバーレイを描画する。前半は 0→1 で暗転、後半は 1→0 で明転する
func (st *FadeTransitionState) Draw(_ w.World, screen *ebiten.Image) error {
	if st.overlay == nil {
		return nil
	}

	var alpha float64
	if st.elapsedMs < st.fadeMs {
		alpha = st.elapsedMs / st.fadeMs
	} else {
		alpha = 2.0 - st.elapsedMs/st.fadeMs
	}
	alpha = max(0, min(1, alpha))

	st.overlay.Fill(color.RGBA{0, 0, 0, uint8(alpha * 255)})
	screen.DrawImage(st.overlay, nil)
	return nil
}
