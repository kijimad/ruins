package ui

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/stretchr/testify/assert"
)

type testMenuProps struct {
	Items []string
}

// setupMenuState はメニュー用のUseStateを登録する
func setupMenuState(store *Store, p testMenuProps) {
	UseState(store, "selected", 0, func(v int, action inputmapper.ActionID) int {
		switch action {
		case inputmapper.ActionMenuUp:
			if v > 0 {
				return v - 1
			}
			return v
		case inputmapper.ActionMenuDown:
			if v < len(p.Items)-1 {
				return v + 1
			}
			return v
		default:
			return v
		}
	})
}

func TestMount_基本的な使用フロー(t *testing.T) {
	t.Parallel()
	// 1. Mountを作成
	mount := NewMount[testMenuProps]()

	// 2. Propsを設定（外部データ）
	props := testMenuProps{Items: []string{"開始", "設定", "終了"}}
	mount.SetProps(props)
	setupMenuState(mount.Store(), props)

	// 3. 初回Updateは常にtrueを返す
	changed := mount.Update()
	assert.True(t, changed, "初回Updateは常にtrue")

	// 4. 変更がなければfalse
	changed = mount.Update()
	assert.False(t, changed, "変更がなければfalse")

	// 5. Dispatchで状態を更新
	mount.Dispatch(inputmapper.ActionMenuDown)

	// 6. 変更があればtrue
	changed = mount.Update()
	assert.True(t, changed, "Dispatch後はtrue")

	// 7. GetStateで状態を取得
	selected, _ := GetState[int](mount, "selected")
	assert.Equal(t, 1, selected)
}

func TestMount_Propsが変わるとUpdateがtrueを返す(t *testing.T) {
	t.Parallel()
	mount := NewMount[testMenuProps]()

	props := testMenuProps{Items: []string{"a"}}
	mount.SetProps(props)
	setupMenuState(mount.Store(), props)
	mount.Update()

	// Propsを変更
	newProps := testMenuProps{Items: []string{"a", "b", "c"}}
	mount.SetProps(newProps)
	setupMenuState(mount.Store(), newProps)
	changed := mount.Update()

	assert.True(t, changed, "Propsが変わればtrue")
}

func TestMount_同じPropsならUpdateがfalseを返す(t *testing.T) {
	t.Parallel()
	mount := NewMount[testMenuProps]()

	props := testMenuProps{Items: []string{"a", "b"}}
	mount.SetProps(props)
	setupMenuState(mount.Store(), props)
	mount.Update()

	// 同じ値のPropsを設定
	sameProps := testMenuProps{Items: []string{"a", "b"}}
	mount.SetProps(sameProps)
	setupMenuState(mount.Store(), sameProps)
	changed := mount.Update()

	assert.False(t, changed, "Propsが同じならfalse")
}

func TestMount_Dispatchで状態が更新される(t *testing.T) {
	t.Parallel()
	mount := NewMount[testMenuProps]()
	props := testMenuProps{Items: []string{"a", "b", "c"}}
	mount.SetProps(props)
	setupMenuState(mount.Store(), props)
	mount.Update()

	// 下に移動
	mount.Dispatch(inputmapper.ActionMenuDown)
	selected, _ := GetState[int](mount, "selected")
	assert.Equal(t, 1, selected)

	// さらに下に移動
	mount.Dispatch(inputmapper.ActionMenuDown)
	selected, _ = GetState[int](mount, "selected")
	assert.Equal(t, 2, selected)

	// 上に移動
	mount.Dispatch(inputmapper.ActionMenuUp)
	selected, _ = GetState[int](mount, "selected")
	assert.Equal(t, 1, selected)
}

func TestMount_Propsの最新値を参照できる(t *testing.T) {
	t.Parallel()
	mount := NewMount[testMenuProps]()

	// 3アイテム
	props := testMenuProps{Items: []string{"a", "b", "c"}}
	mount.SetProps(props)
	setupMenuState(mount.Store(), props)
	mount.Update()

	mount.Dispatch(inputmapper.ActionMenuDown)
	mount.Dispatch(inputmapper.ActionMenuDown)
	mount.Dispatch(inputmapper.ActionMenuDown) // 上限で止まる
	mount.Update()

	selected, _ := GetState[int](mount, "selected")
	assert.Equal(t, 2, selected, "3アイテムなので最大2")

	// 5アイテムに増やす
	newProps := testMenuProps{Items: []string{"a", "b", "c", "d", "e"}}
	mount.SetProps(newProps)
	setupMenuState(mount.Store(), newProps) // 新しいPropsでreducerが再登録される
	mount.Update()

	mount.Dispatch(inputmapper.ActionMenuDown)
	mount.Dispatch(inputmapper.ActionMenuDown)
	mount.Update()

	selected, _ = GetState[int](mount, "selected")
	assert.Equal(t, 4, selected, "5アイテムなので最大4まで移動可能")
}

func TestGetState_存在しないキーはfalseを返す(t *testing.T) {
	t.Parallel()
	mount := NewMount[testMenuProps]()

	_, ok := GetState[int](mount, "notfound")
	assert.False(t, ok)
}

func TestGetState_型が違うとfalseを返す(t *testing.T) {
	t.Parallel()
	mount := NewMount[testMenuProps]()
	props := testMenuProps{}
	mount.SetProps(props)
	setupMenuState(mount.Store(), props)
	mount.Update()

	_, ok := GetState[string](mount, "selected")
	assert.False(t, ok, "intをstringで取得しようとするとfalse")
}
