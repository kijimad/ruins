package ui

// UseRef は再レンダリングしても値が保持される参照を提供する
// React の useRef に相当する
// init は初回のみ呼ばれ、以降はキャッシュされた値を返す
func UseRef[T any](store *Store, key string, init func() T) T {
	if _, ok := store.refs[key]; !ok {
		store.refs[key] = init()
	}
	return store.refs[key].(T)
}

// GetRef は登録済みの参照を取得する
// 存在しない場合はゼロ値と false を返す
func GetRef[T any](store *Store, key string) (T, bool) {
	if val, ok := store.refs[key]; ok {
		return val.(T), true
	}
	var zero T
	return zero, false
}
