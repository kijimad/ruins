package hooks

// UseRef は再レンダリングしても値が保持される参照を提供する
// React の useRef に相当する
// init は初回のみ呼ばれ、以降はキャッシュされた値を返す
func UseRef[T any](store *Store, key string, init func() T) T {
	if _, ok := store.refs[key]; !ok {
		store.refs[key] = init()
	}
	v, ok := store.refs[key].(T)
	if !ok {
		panic("hooks: 参照の型が登録時と一致しません: key=" + key)
	}
	return v
}

// GetRef は登録済みの参照を取得する
// 存在しない場合や型が一致しない場合はゼロ値と false を返す
func GetRef[T any](store *Store, key string) (T, bool) {
	if val, ok := store.refs[key]; ok {
		if typed, ok := val.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}
