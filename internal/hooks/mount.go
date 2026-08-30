package hooks

import (
	"reflect"

	"github.com/kijimaD/ruins/internal/inputmapper"
)

// Mount はProps + Stateを管理し、変更を検出する
// 描画は担当しない。描画はアプリケーション層の責務である
type Mount[Props any] struct {
	props Props
	store *Store
	// dirty は props 変化と初回描画の要因を持つ。store 状態の変化は store.Version で別に追う。
	dirty            bool
	lastStoreVersion uint64
}

// NewMount は新しいMountを生成する
func NewMount[Props any]() *Mount[Props] {
	return &Mount[Props]{
		store: NewStore(),
		dirty: true, // 初回は必ず描画する
	}
}

// SetProps は外部からPropsを設定する
// 値が変わった場合はdirtyフラグを立てる
func (m *Mount[Props]) SetProps(props Props) {
	if !reflect.DeepEqual(m.props, props) {
		m.dirty = true
	}
	m.props = props
}

// GetProps は現在のPropsを返す
func (m *Mount[Props]) GetProps() Props {
	return m.props
}

// Store はStoreを返す
// UseStateやUseTabMenuを呼び出すために使用する
func (m *Mount[Props]) Store() *Store {
	return m.store
}

// Dispatch は全てのStateにActionを送る。実際に状態が変われば store.Version が動き、
// 次の Update が再描画と判定する。手動で dirty を立てる必要はない
func (m *Mount[Props]) Dispatch(action inputmapper.ActionID) {
	m.store.Dispatch(action)
}

// GetState は指定したキーのStateを取得する
func GetState[T any, Props any](m *Mount[Props], key string) (T, bool) {
	v, ok := m.store.states[key]
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

// Update は変更の有無を返す
// 初回は常にtrue、以降はpropsまたはstateが変わった場合にtrueを返す。
// store 状態の変化は version の差で検知するので、Dispatch でも SetTab でも書き込み経路に依らず拾える
func (m *Mount[Props]) Update() bool {
	changed := m.dirty || m.store.Version() != m.lastStoreVersion
	m.dirty = false
	m.lastStoreVersion = m.store.Version()
	return changed
}
