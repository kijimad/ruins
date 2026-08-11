package messagewindow

import (
	"testing"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

func TestView_GetCurrentPage(t *testing.T) {
	t.Parallel()

	t.Run("1ページ目は1を返す", func(t *testing.T) {
		t.Parallel()
		view := newView(tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: make([]item, 10)}},
			ItemsPerPage: 3,
		}, w.World{})
		view.SetState(viewState{TabIndex: 0, ItemIndex: 0})
		assert.Equal(t, 1, view.GetCurrentPage())
	})

	t.Run("2ページ目は2を返す", func(t *testing.T) {
		t.Parallel()
		view := newView(tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: make([]item, 10)}},
			ItemsPerPage: 3,
		}, w.World{})
		view.SetState(viewState{TabIndex: 0, ItemIndex: 4})
		assert.Equal(t, 2, view.GetCurrentPage())
	})
}
