package ui

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/stretchr/testify/assert"
)

// Storeは状態管理を担当する
// UseStateで状態を登録し、dispatchでアクションを送ると状態が更新される

func TestUseState_初回呼び出しで初期値を返す(t *testing.T) {
	t.Parallel()
	store := NewStore()

	// UseState(store, キー, 初期値, 更新関数) → 現在の値を返す
	selected := UseState(store, "selected", 0, func(v int, _ inputmapper.ActionID) int {
		return v // 更新関数はdispatch時に呼ばれる
	})

	assert.Equal(t, 0, selected, "初回は初期値が返る")
}

func TestUseState_2回目以降は既存の値を返す(t *testing.T) {
	t.Parallel()
	store := NewStore()

	// 1回目: 初期値0で登録
	UseState(store, "selected", 0, func(v int, _ inputmapper.ActionID) int {
		return v
	})

	// 2回目: 初期値999を渡しても無視される
	selected := UseState(store, "selected", 999, func(v int, _ inputmapper.ActionID) int {
		return v
	})

	assert.Equal(t, 0, selected, "2回目以降は初期値が無視され、既存の値が返る")
}

func TestDispatch_更新関数が呼ばれて状態が変わる(t *testing.T) {
	t.Parallel()
	store := NewStore()

	// カウンターを登録。どのアクションでも+1する
	UseState(store, "count", 0, func(v int, _ inputmapper.ActionID) int {
		return v + 1
	})

	// dispatch前
	assert.Equal(t, 0, store.states["count"])

	// dispatchするとupdate関数が呼ばれる
	store.Dispatch(inputmapper.ActionMenuUp)

	assert.Equal(t, 1, store.states["count"], "dispatchで更新関数が実行される")
}

func TestDispatch_アクションに応じて異なる処理ができる(t *testing.T) {
	t.Parallel()
	store := NewStore()

	// アクションに応じて増減するカウンター
	UseState(store, "index", 5, func(v int, action inputmapper.ActionID) int {
		switch action {
		case inputmapper.ActionMenuUp:
			return v - 1
		case inputmapper.ActionMenuDown:
			return v + 1
		default:
			return v
		}
	})

	store.Dispatch(inputmapper.ActionMenuDown)
	assert.Equal(t, 6, store.states["index"], "Downで+1")

	store.Dispatch(inputmapper.ActionMenuUp)
	assert.Equal(t, 5, store.states["index"], "Upで-1")

	store.Dispatch(inputmapper.ActionMenuLeft)
	assert.Equal(t, 5, store.states["index"], "関係ないアクションでは変化なし")
}

func TestDispatch_複数の状態が同時に更新される(t *testing.T) {
	t.Parallel()
	store := NewStore()

	// 2つの独立した状態を登録
	UseState(store, "tabIndex", 0, func(v int, action inputmapper.ActionID) int {
		if action == inputmapper.ActionMenuRight {
			return v + 1
		}
		return v
	})
	UseState(store, "itemIndex", 0, func(v int, action inputmapper.ActionID) int {
		if action == inputmapper.ActionMenuDown {
			return v + 1
		}
		return v
	})

	// 1回のdispatchで全ての状態の更新関数が呼ばれる
	store.Dispatch(inputmapper.ActionMenuRight)

	assert.Equal(t, 1, store.states["tabIndex"], "tabIndexはRightで更新")
	assert.Equal(t, 0, store.states["itemIndex"], "itemIndexはRightでは変化なし")
}

func TestUseState_更新関数は毎回再登録される(t *testing.T) {
	t.Parallel()
	store := NewStore()

	// 1回目: 上限を3として登録
	limit := 3
	UseState(store, "index", 0, func(v int, _ inputmapper.ActionID) int {
		if v < limit {
			return v + 1
		}
		return v
	})

	store.Dispatch(inputmapper.ActionMenuDown)
	store.Dispatch(inputmapper.ActionMenuDown)
	store.Dispatch(inputmapper.ActionMenuDown)
	store.Dispatch(inputmapper.ActionMenuDown)
	assert.Equal(t, 3, store.states["index"], "上限が3なので3で止まる")

	// 2回目: 上限を5に変更して再登録
	limit = 5
	UseState(store, "index", 0, func(v int, _ inputmapper.ActionID) int {
		if v < limit {
			return v + 1
		}
		return v
	})

	store.Dispatch(inputmapper.ActionMenuDown)
	store.Dispatch(inputmapper.ActionMenuDown)
	assert.Equal(t, 5, store.states["index"], "再登録後は新しい上限が適用される")
}
