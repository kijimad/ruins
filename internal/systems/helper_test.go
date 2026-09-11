package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeSystems は全システムの登録を固定する。各システムを String() のキーで
// updaters/renderers へ登録するので、代表的なシステムの存在と、キーが String() と一致する
// 不変条件、Updater と Renderer 両方へ登録される特殊システムを検証する。
// WithUI は HUDRenderingSystem など UI リソースを要するシステムの初期化のため。
func TestInitializeSystems(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	updaters, renderers := InitializeSystems(world)

	t.Run("代表的なUpdaterが登録される", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, updaters)
		for _, name := range []string{"CameraSystem", "AnimationSystem", "TurnSystem", "TemperatureSystem", "VisionSystem", "AuctionSystem"} {
			assert.Contains(t, updaters, name, "%s が登録される", name)
		}
	})

	t.Run("代表的なRendererが登録される", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, renderers)
		assert.Contains(t, renderers, "RenderSpriteSystem", "スプライト描画が登録される")
	})

	t.Run("HUDとVisualEffectはUpdaterとRendererの両方へ登録される", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"HUDRenderingSystem", "VisualEffectSystem"} {
			assert.Contains(t, updaters, name, "%s は Updater にも登録される", name)
			assert.Contains(t, renderers, name, "%s は Renderer にも登録される", name)
		}
	})

	t.Run("マップのキーは対応するStringと一致する", func(t *testing.T) {
		t.Parallel()
		for name, u := range updaters {
			assert.Equal(t, name, u.String(), "Updater のキーは String() と一致する")
		}
		for name, r := range renderers {
			assert.Equal(t, name, r.String(), "Renderer のキーは String() と一致する")
		}
	})
}
