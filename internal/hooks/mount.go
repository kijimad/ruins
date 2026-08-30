package hooks

import "github.com/kijimaD/ruins/internal/inputmapper"

// mountPropsKey は Mount が props を Store に保持するときの予約キー。props も view を決める状態なので
// Store の version 追跡に載せ、変更検知を1系統へ集約する。利用側の UseState キーと衝突しないよう
// 制御文字始まりの予約名にする。
const mountPropsKey = "\x00mount.props"

// Mount はProps + Stateを管理し、変更を検出する。
// 描画は担当しない。描画はアプリケーション層の責務である。
// props も store の予約キーに載せるので、変更検知は store.Version の差だけで済む。
type Mount[Props any] struct {
	store            *Store
	lastStoreVersion uint64
}

// NewMount は新しいMountを生成する
func NewMount[Props any]() *Mount[Props] {
	return &Mount[Props]{
		store: NewStore(),
		// 初回は必ず描画させる。実在しない version にしておき、最初の Update で必ず差を出す
		lastStoreVersion: ^uint64(0),
	}
}

// SetProps は外部からPropsを設定する。値が実際に変わったときだけ store の version が動く
func (m *Mount[Props]) SetProps(props Props) {
	m.store.set(mountPropsKey, props)
}

// GetProps は現在のPropsを返す。未設定ならゼロ値
func (m *Mount[Props]) GetProps() Props {
	v, _ := GetStoreState[Props](m.store, mountPropsKey)
	return v
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
	return GetStoreState[T](m.store, key)
}

// Update は前回からの変更の有無を返す。props も store 状態も store.Version に集約されるので、
// version の差だけで判定できる。Dispatch でも SetTab でも書き込み経路に依らず拾える。
// 初回は sentinel 初期化により必ず true になる。
func (m *Mount[Props]) Update() bool {
	v := m.store.Version()
	changed := v != m.lastStoreVersion
	m.lastStoreVersion = v
	return changed
}
