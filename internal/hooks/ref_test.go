package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUseRef_InitialValue(t *testing.T) {
	t.Parallel()

	store := NewStore()
	initCallCount := 0

	value := UseRef(store, "test", func() int {
		initCallCount++
		return 42
	})

	assert.Equal(t, 42, value, "初期値が返される")
	assert.Equal(t, 1, initCallCount, "init関数は1回呼ばれる")
}

func TestUseRef_CachedValue(t *testing.T) {
	t.Parallel()

	store := NewStore()
	initCallCount := 0

	// 1回目の呼び出し
	value1 := UseRef(store, "test", func() int {
		initCallCount++
		return 42
	})

	// 2回目の呼び出し
	value2 := UseRef(store, "test", func() int {
		initCallCount++
		return 100 // 異なる値を返す init を渡しても無視される
	})

	assert.Equal(t, 42, value1, "初期値が返される")
	assert.Equal(t, 42, value2, "キャッシュされた値が返される")
	assert.Equal(t, 1, initCallCount, "init関数は最初の1回だけ呼ばれる")
}

func TestUseRef_MultipleKeys(t *testing.T) {
	t.Parallel()

	store := NewStore()

	value1 := UseRef(store, "key1", func() string { return "hello" })
	value2 := UseRef(store, "key2", func() string { return "world" })

	assert.Equal(t, "hello", value1, "key1の値")
	assert.Equal(t, "world", value2, "key2の値")
}

func TestUseRef_PointerType(t *testing.T) {
	t.Parallel()

	type Widget struct {
		Name string
	}

	store := NewStore()

	// ポインタ型でも動作する
	widget := UseRef(store, "widget", func() *Widget {
		return &Widget{Name: "button"}
	})

	assert.Equal(t, "button", widget.Name, "初期値が設定される")

	// 値を変更
	widget.Name = "textInput"

	// 再取得すると変更された値が返される
	widget2 := UseRef(store, "widget", func() *Widget {
		return &Widget{Name: "should not be used"}
	})

	assert.Equal(t, "textInput", widget2.Name, "変更された値が保持される")
	assert.Same(t, widget, widget2, "同じポインタが返される")
}

func TestGetRef_NotExists(t *testing.T) {
	t.Parallel()

	store := NewStore()

	value, ok := GetRef[int](store, "notexists")

	assert.False(t, ok, "存在しないキーは false を返す")
	assert.Equal(t, 0, value, "ゼロ値が返される")
}

func TestGetRef_Exists(t *testing.T) {
	t.Parallel()

	store := NewStore()

	// UseRef で登録
	UseRef(store, "test", func() int { return 42 })

	// GetRef で取得
	value, ok := GetRef[int](store, "test")

	assert.True(t, ok, "存在するキーは true を返す")
	assert.Equal(t, 42, value, "登録した値が返される")
}

func TestGetRef_PointerType(t *testing.T) {
	t.Parallel()

	type Widget struct {
		Name string
	}

	store := NewStore()

	// 存在しない場合
	widget, ok := GetRef[*Widget](store, "widget")
	assert.False(t, ok, "存在しないキーは false を返す")
	assert.Nil(t, widget, "ポインタ型のゼロ値は nil")

	// UseRef で登録
	UseRef(store, "widget", func() *Widget {
		return &Widget{Name: "button"}
	})

	// GetRef で取得
	widget, ok = GetRef[*Widget](store, "widget")
	assert.True(t, ok, "存在するキーは true を返す")
	assert.Equal(t, "button", widget.Name, "登録した値が返される")
}
