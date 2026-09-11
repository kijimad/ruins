package overlay

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewComparison_複数内容を開く は、複数の詳細内容を返す provide を受けた Detail が
// 開けること、中身が無ければ開かないことを固定する。
func TestNewComparison_複数内容を開く(t *testing.T) {
	t.Parallel()

	d := NewComparison(func(_ w.World) ([]DetailContent, bool) {
		return []DetailContent{{Name: "A"}, {Name: "B"}}, true
	})
	assert.False(t, d.Active())

	d.Open(w.World{})
	assert.True(t, d.Active(), "内容があれば開く")

	empty := NewComparison(func(_ w.World) ([]DetailContent, bool) { return nil, false })
	empty.Open(w.World{})
	assert.False(t, empty.Active(), "内容が無ければ開かない")
}

// TestDetailHandleInput_ページ送りと閉じる は、詳細モーダルの入力を注入した供給源で駆動し、
// 前後のページ送り・端での停止・キャンセルでの閉じるを固定する。
func TestDetailHandleInput_ページ送りと閉じる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	var action inputmapper.ActionID
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return action, true }

	// 3ページになる行数。定数を直接使い、変更に自動追随させる
	rows := make([]entityspec.SpecRow, detailRowsPerPage*2+1)
	d := NewComparison(func(_ w.World) ([]DetailContent, bool) {
		return []DetailContent{{Name: "A", Rows: rows}, {Name: "B"}}, true
	})
	d.Open(world)
	require.True(t, d.Active())
	assert.Equal(t, 0, d.page)

	action = inputmapper.ActionMenuRight
	require.NoError(t, d.HandleInput(world))
	assert.Equal(t, 1, d.page, "右で次ページ")

	action = inputmapper.ActionMenuTabNext
	require.NoError(t, d.HandleInput(world))
	assert.Equal(t, 2, d.page, "TabNextで次ページ")

	action = inputmapper.ActionMenuRight
	require.NoError(t, d.HandleInput(world))
	assert.Equal(t, 2, d.page, "最終ページでは止まる")

	action = inputmapper.ActionMenuTabPrev
	require.NoError(t, d.HandleInput(world))
	assert.Equal(t, 1, d.page, "TabPrevで前ページ")

	action = inputmapper.ActionMenuCancel
	require.NoError(t, d.HandleInput(world))
	assert.False(t, d.Active(), "キャンセルで閉じる")
}

// TestDetailHandleInput_先頭で左は止まりSelectで閉じる は、先頭ページでの左入力が止まること、
// 決定入力で閉じることを固定する。
func TestDetailHandleInput_先頭で左は止まりSelectで閉じる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	var action inputmapper.ActionID
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return action, true }

	d := NewComparison(func(_ w.World) ([]DetailContent, bool) {
		return []DetailContent{{Name: "A"}}, true
	})
	d.Open(world)

	action = inputmapper.ActionMenuLeft
	require.NoError(t, d.HandleInput(world))
	assert.Equal(t, 0, d.page, "先頭ページで左は止まる")

	action = inputmapper.ActionMenuSelect
	require.NoError(t, d.HandleInput(world))
	assert.False(t, d.Active(), "決定で閉じる")
}

// TestDetailHandleInput_入力が無ければ開いたまま は、供給源が入力なしを返すとき状態が
// 変わらないことを固定する。
func TestDetailHandleInput_入力が無ければ開いたまま(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return "", false }

	d := NewComparison(func(_ w.World) ([]DetailContent, bool) {
		return []DetailContent{{Name: "A"}}, true
	})
	d.Open(world)

	require.NoError(t, d.HandleInput(world))
	assert.True(t, d.Active(), "入力が無ければ開いたまま")
}
