package tabmenu

import (
	"testing"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

func TestView_GetCurrentPage(t *testing.T) {
	t.Parallel()

	t.Run("1ページ目は1を返す", func(t *testing.T) {
		t.Parallel()
		view := NewView(Config{
			Tabs:         []TabItem{{ID: "t1", Items: make([]Item, 10)}},
			ItemsPerPage: 3,
		}, w.World{})
		view.SetState(ViewState{TabIndex: 0, ItemIndex: 0})
		assert.Equal(t, 1, view.GetCurrentPage())
	})

	t.Run("2ページ目は2を返す", func(t *testing.T) {
		t.Parallel()
		view := NewView(Config{
			Tabs:         []TabItem{{ID: "t1", Items: make([]Item, 10)}},
			ItemsPerPage: 3,
		}, w.World{})
		view.SetState(ViewState{TabIndex: 0, ItemIndex: 4})
		assert.Equal(t, 2, view.GetCurrentPage())
	})
}

func TestView_UpdateTabs_タブを置き換える(t *testing.T) {
	t.Parallel()
	view := NewView(Config{
		Tabs: []TabItem{{ID: "old", Label: "旧タブ"}},
	}, w.World{})

	newTabs := []TabItem{{ID: "new", Label: "新タブ"}}
	view.UpdateTabs(newTabs)

	assert.Equal(t, newTabs, view.config.Tabs)
}
