package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &SettingsMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)

	// 現状は「戻る」のみ。将来の設定項目を追加する土台
	require.Len(t, props.Items, 1)
	assert.Equal(t, "戻る", props.Items[0].Label)
	assert.Equal(t, es.TransPop, props.Items[0].Transition.Type)
}
