package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	p := New(5, 20, 10)

	assert.Equal(t, 5, p.ItemIndex)
	assert.Equal(t, 0, p.Page)
	assert.Equal(t, 20, p.ItemCount)
	assert.Equal(t, 10, p.ItemsPerPage)

	// ページ計算の確認
	p2 := New(15, 20, 10)
	assert.Equal(t, 1, p2.Page)
}

func TestPagination_GetCurrentPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		page     int
		expected int
	}{
		{"ページ0は1を返す", 0, 1},
		{"ページ1は2を返す", 1, 2},
		{"ページ5は6を返す", 5, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := Pagination{Page: tt.page}
			assert.Equal(t, tt.expected, p.GetCurrentPage())
		})
	}
}

func TestPagination_GetTotalPages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		itemCount    int
		itemsPerPage int
		expected     int
	}{
		{"10アイテム、5/ページ = 2ページ", 10, 5, 2},
		{"11アイテム、5/ページ = 3ページ", 11, 5, 3},
		{"5アイテム、5/ページ = 1ページ", 5, 5, 1},
		{"0アイテム = 1ページ", 0, 5, 1},
		{"itemsPerPage=0 = 1ページ", 10, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := Pagination{ItemCount: tt.itemCount, ItemsPerPage: tt.itemsPerPage}
			assert.Equal(t, tt.expected, p.GetTotalPages())
		})
	}
}

func TestPagination_GetVisibleRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		page          int
		itemCount     int
		itemsPerPage  int
		expectedStart int
		expectedEnd   int
	}{
		{"ページ0、10アイテム、5/ページ", 0, 10, 5, 0, 5},
		{"ページ1、10アイテム、5/ページ", 1, 10, 5, 5, 10},
		{"最後のページが半端", 1, 8, 5, 5, 8},
		{"itemsPerPage=0は全て返す", 0, 10, 0, 0, 10},
		{"範囲外ページは空", 5, 10, 5, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := Pagination{
				Page:         tt.page,
				ItemCount:    tt.itemCount,
				ItemsPerPage: tt.itemsPerPage,
			}
			start, end := p.GetVisibleRange()
			assert.Equal(t, tt.expectedStart, start)
			assert.Equal(t, tt.expectedEnd, end)
		})
	}
}

func TestPagination_HasPreviousPage(t *testing.T) {
	t.Parallel()

	assert.False(t, Pagination{Page: 0}.HasPreviousPage())
	assert.True(t, Pagination{Page: 1}.HasPreviousPage())
	assert.True(t, Pagination{Page: 5}.HasPreviousPage())
}

func TestPagination_HasNextPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		page         int
		itemCount    int
		itemsPerPage int
		expected     bool
	}{
		{"次ページあり", 0, 10, 5, true},
		{"最後のページ", 1, 10, 5, false},
		{"itemsPerPage=0", 0, 10, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := Pagination{
				Page:         tt.page,
				ItemCount:    tt.itemCount,
				ItemsPerPage: tt.itemsPerPage,
			}
			assert.Equal(t, tt.expected, p.HasNextPage())
		})
	}
}

func TestPagination_IsEnabled(t *testing.T) {
	t.Parallel()

	assert.False(t, Pagination{ItemCount: 5, ItemsPerPage: 10}.IsEnabled())
	assert.True(t, Pagination{ItemCount: 15, ItemsPerPage: 10}.IsEnabled())
}

func TestPagination_GetIndicatorText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		page         int
		itemCount    int
		itemsPerPage int
		expected     string
	}{
		{"1ページのみは空文字", 0, 5, 10, ""},
		{"最初のページ", 0, 15, 5, "  1/3 ↓"},
		{"中間ページ", 1, 15, 5, "↑ 2/3 ↓"},
		{"最後のページ", 2, 15, 5, "↑ 3/3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := Pagination{
				Page:         tt.page,
				ItemCount:    tt.itemCount,
				ItemsPerPage: tt.itemsPerPage,
			}
			assert.Equal(t, tt.expected, p.GetIndicatorText())
		})
	}
}

func TestSliceVisible(t *testing.T) {
	t.Parallel()

	items := []string{"a", "b", "c", "d", "e", "f", "g"}
	p := Pagination{Page: 1, ItemCount: 7, ItemsPerPage: 3}

	visible := SliceVisible(items, p)
	assert.Equal(t, []string{"d", "e", "f"}, visible)
}

func TestVisibleEntries(t *testing.T) {
	t.Parallel()

	items := []string{"a", "b", "c", "d", "e"}
	p := Pagination{Page: 1, ItemCount: 5, ItemsPerPage: 3}

	result := VisibleEntries(items, p)
	assert.Len(t, result, 2)
	assert.Equal(t, 3, result[0].Index)
	assert.Equal(t, "d", result[0].Item)
	assert.Equal(t, 4, result[1].Index)
	assert.Equal(t, "e", result[1].Item)
}
