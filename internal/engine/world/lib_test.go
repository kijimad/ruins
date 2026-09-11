package world

import (
	"errors"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	c "github.com/kijimaD/ruins/internal/engine/components"
	r "github.com/kijimaD/ruins/internal/engine/resources"
	gr "github.com/kijimaD/ruins/internal/resources"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitGeneric(t *testing.T) {
	t.Parallel()
	t.Run("型安全なInitGenericが動作する", func(t *testing.T) {
		t.Parallel()
		gameComponents := &gc.Components{}
		gameResources := &gr.Resources{}

		world, err := InitGeneric(gameComponents, gameResources)

		require.NoError(t, err)
		assert.NotNil(t, world.ECS)
		assert.NotNil(t, world.Components)
		assert.NotNil(t, world.Resources)

		// 型安全性の確認
		assert.IsType(t, &gc.Components{}, world.Components.Game)
		assert.IsType(t, &gr.Resources{}, world.Resources.Game)
	})

	t.Run("型安全性が保たれている", func(t *testing.T) {
		t.Parallel()
		gameComponents := &gc.Components{}
		gameResources := &gr.Resources{}

		world, err := InitGeneric(gameComponents, gameResources)

		require.NoError(t, err)
		// 型アサーションが不要で、直接アクセスできる
		assert.NotNil(t, world.Components.Game.Position)
		assert.NotNil(t, world.Resources.Game.ScreenDimensions)
	})
}

var errInit = errors.New("init failed")

// インターフェース充足を明示する。InitGeneric の型引数が要求する初期化子であることを
// コンパイル時に固定する。
var (
	_ c.ComponentInitializer = failComponents{}
	_ c.ComponentInitializer = okComponents{}
	_ r.ResourceInitializer  = failResources{}
	_ r.ResourceInitializer  = okResources{}
)

// failComponents は InitializeComponents で失敗する初期化子。
type failComponents struct{}

func (failComponents) InitializeComponents(*ecs.World) error { return errInit }

// okComponents は成功する初期化子。リソース側の失敗を試すために使う。
type okComponents struct{}

func (okComponents) InitializeComponents(*ecs.World) error { return nil }

// failResources は InitializeResources で失敗する初期化子。
type failResources struct{}

func (failResources) InitializeResources() error { return errInit }

// okResources は成功する初期化子。コンポーネント側の失敗を試すために使う。
type okResources struct{}

func (okResources) InitializeResources() error { return nil }

// TestInitGeneric_コンポーネント初期化の失敗を伝播する は、コンポーネント初期化で
// エラーが出たとき InitGeneric がそれを包んで返すことを固定する。
func TestInitGeneric_コンポーネント初期化の失敗を伝播する(t *testing.T) {
	t.Parallel()

	_, err := InitGeneric(failComponents{}, okResources{})
	require.ErrorIs(t, err, errInit)
}

// TestInitGeneric_リソース初期化の失敗を伝播する は、リソース初期化で
// エラーが出たとき InitGeneric がそれを包んで返すことを固定する。
func TestInitGeneric_リソース初期化の失敗を伝播する(t *testing.T) {
	t.Parallel()

	_, err := InitGeneric(okComponents{}, failResources{})
	require.ErrorIs(t, err, errInit)
}
