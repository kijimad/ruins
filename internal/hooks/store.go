package hooks

import (
	"reflect"

	"github.com/kijimaD/ruins/internal/inputmapper"
)

// Reducer は個別の状態を更新する関数の型
type Reducer func(state any, action inputmapper.ActionID) any

// Store は状態とreducerを保持する
type Store struct {
	states   map[string]any
	reducers map[string]Reducer
	version  uint64 // 状態が実際に変わるたびに増える。Mount が再描画要否をこれで判定する
}

// NewStore は新しいStoreを生成する
func NewStore() *Store {
	return &Store{
		states:   make(map[string]any),
		reducers: make(map[string]Reducer),
	}
}

// set は状態を書き、値が実際に変わったときだけ version を上げる。states への書き込みは
// すべてここを通し、変更検知を1点へ集約する。Dispatch と SetTab で経路が分かれても、
// 実際に変わったかで一貫して dirty を導ける。毎フレーム同値を書く UseTabMenu では版が動かない。
func (store *Store) set(key string, val any) {
	if old, ok := store.states[key]; ok && reflect.DeepEqual(old, val) {
		return
	}
	store.states[key] = val
	store.version++
}

// Version は現在の状態版を返す。値が実際に変わるたびに増える
func (store *Store) Version() uint64 { return store.version }

// UseState は状態を取得・登録する
// keyで状態を識別し、初回呼び出し時にinitで初期化する
// reducer関数はDispatch時に呼ばれ、状態を更新する
// reducerは毎回再登録される。これによりProps変化時に最新のクロージャが使われる
func UseState[T any](store *Store, key string, init T, reducer func(T, inputmapper.ActionID) T) T {
	if _, ok := store.states[key]; !ok {
		store.set(key, init)
	}
	// 毎回reducerを再登録して最新のクロージャを反映する
	store.reducers[key] = func(s any, a inputmapper.ActionID) any {
		typed, ok := s.(T)
		if !ok {
			panic("hooks: state type does not match reducer type argument: key=" + key)
		}
		return reducer(typed, a)
	}
	v, ok := store.states[key].(T)
	if !ok {
		panic("hooks: state type does not match registration: key=" + key)
	}
	return v
}

// Dispatch は全てのStateにActionを送る
func (store *Store) Dispatch(action inputmapper.ActionID) {
	for key, reducer := range store.reducers {
		store.set(key, reducer(store.states[key], action))
	}
}

// GetStoreState は指定キーの状態を取得する。Mount を経由せずに直接取得できる
func GetStoreState[T any](store *Store, key string) (T, bool) {
	v, ok := store.states[key]
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typed, true
}
