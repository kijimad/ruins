package hooks

import "github.com/kijimaD/ruins/internal/inputmapper"

// mountPropsKey は props を Store に載せるときの予約キー。props も version 追跡に乗せ変更検知を
// 1系統にする。UseState キーと衝突しないよう制御文字始まりにする。
const mountPropsKey = "\x00mount.props"

// Mount はProps + Stateを管理し変更を検出する。描画は担当しない。
// props も store に載るので、変更検知は store.Version の差だけで済む。
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

// Dispatch は全ての State に Action を送る。状態が変われば store.Version が動き次の Update が拾う
func (m *Mount[Props]) Dispatch(action inputmapper.ActionID) {
	m.store.Dispatch(action)
}

// GetState は指定したキーのStateを取得する
func GetState[T any, Props any](m *Mount[Props], key string) (T, bool) {
	return GetStoreState[T](m.store, key)
}

// Update は前回からの変更の有無を返す。props も store 状態も store.Version に集約されるので
// 版差だけで判定する。初回は sentinel 初期化で必ず true になる。
func (m *Mount[Props]) Update() bool {
	v := m.store.Version()
	changed := v != m.lastStoreVersion
	m.lastStoreVersion = v
	return changed
}
