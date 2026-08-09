package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComponentDebugState_ナビゲーションアクションでpanicしない は、単一タブのメニューでも
// タブ移動などのナビゲーションアクションが届くため、それらでエラーを返さないことを検証する。
// menu_tab_next で "unknown action" エラーになりクラッシュした回帰を防ぐ
func TestComponentDebugState_ナビゲーションアクションでpanicしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	state, err := NewComponentDebugState()
	require.NoError(t, err)
	cd, ok := state.(*ComponentDebugState)
	require.True(t, ok)

	navActions := []inputmapper.ActionID{
		inputmapper.ActionMenuTabNext,
		inputmapper.ActionMenuTabPrev,
		inputmapper.ActionMenuLeft,
		inputmapper.ActionMenuRight,
		inputmapper.ActionMenuUp,
		inputmapper.ActionMenuDown,
	}
	for _, action := range navActions {
		_, err := cd.DoAction(world, action)
		assert.NoError(t, err, "ナビゲーションアクション %s でエラーを返さない", action)
	}
}
