package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeSystems は全システムの登録を固定する。各システムを String() のキーで
// updaters/renderers へ登録するので、代表的なシステムの存在と、キーが String() と一致する
// 不変条件を検証する。
func TestInitializeSystems(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())

	updaters, renderers := InitializeSystems(world)

	require.NotEmpty(t, updaters, "Updater が登録される")
	require.NotEmpty(t, renderers, "Renderer が登録される")

	// 代表的な Updater が登録されている
	for _, name := range []string{"CameraSystem", "AnimationSystem", "TurnSystem", "TemperatureSystem", "VisionSystem", "AuctionSystem"} {
		assert.Contains(t, updaters, name, "%s が登録される", name)
	}

	// マップのキーは対応するシステムの String() と一致する
	for name, u := range updaters {
		assert.Equal(t, name, u.String(), "Updater のキーは String() と一致する")
	}
	for name, r := range renderers {
		assert.Equal(t, name, r.String(), "Renderer のキーは String() と一致する")
	}
}
