package ui

import "github.com/kijimaD/ruins/internal/inputmapper"

// Reducer は個別の状態を更新する関数の型
type Reducer func(state any, action inputmapper.ActionID) any

// Store は状態とreducerを保持する
type Store struct {
	states   map[string]any
	reducers map[string]Reducer
}

// NewStore は新しいStoreを生成する
func NewStore() *Store {
	return &Store{
		states:   make(map[string]any),
		reducers: make(map[string]Reducer),
	}
}

// UseState は状態を取得・登録する
// keyで状態を識別し、初回呼び出し時にinitで初期化する
// reducer関数はDispatch時に呼ばれ、状態を更新する
// reducerは毎回再登録される。これによりProps変化時に最新のクロージャが使われる
func UseState[T any](store *Store, key string, init T, reducer func(T, inputmapper.ActionID) T) T {
	if _, ok := store.states[key]; !ok {
		store.states[key] = init
	}
	// 毎回reducerを再登録して最新のクロージャを反映する
	store.reducers[key] = func(s any, a inputmapper.ActionID) any {
		return reducer(s.(T), a)
	}
	return store.states[key].(T)
}

// dispatch は全てのStateにActionを送る
func (store *Store) dispatch(action inputmapper.ActionID) {
	for key, reducer := range store.reducers {
		store.states[key] = reducer(store.states[key], action)
	}
}
