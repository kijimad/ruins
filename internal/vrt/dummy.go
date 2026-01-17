package vrt

import (
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
)

// dummyState は複数stateをpushした状態を作り出す。具体stateを介してしか操作できないので必要
type dummyState struct {
	es.BaseState[w.World]
	states []es.State[w.World]
}

func (st *dummyState) OnStart(world w.World) error {
	return st.states[0].OnStart(world)
}

func (st *dummyState) OnStop(world w.World) error {
	return st.states[0].OnStop(world)
}

func (st *dummyState) OnPause(world w.World) error {
	return st.states[0].OnPause(world)
}

func (st *dummyState) OnResume(world w.World) error {
	return st.states[0].OnResume(world)
}

func (st *dummyState) Update(world w.World) (es.Transition[w.World], error) {
	// 2番目以降のstateがある場合、最初のUpdate時にpushする
	if len(st.states) > 1 {
		factories := make([]es.StateFactory[w.World], len(st.states)-1)
		for i, state := range st.states[1:] {
			capturedState := state
			factories[i] = func() es.State[w.World] { return capturedState }
		}
		// pushが完了したらstatesを消費
		st.states = st.states[:1]
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: factories}, nil
	}

	// BaseStateの共通処理を使用
	trans := st.ConsumeTransition()
	if trans.Type != es.TransNone {
		return trans, nil
	}

	return st.states[0].Update(world)
}

func (st *dummyState) Draw(world w.World, screen *ebiten.Image) error {
	return st.states[0].Draw(world, screen)
}
