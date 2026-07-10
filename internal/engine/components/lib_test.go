package components

import (
	"errors"
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用のコンポーネント
type testSliceData struct{ V int }
type testNullData struct{}

type TestComponents struct {
	TestSlice *ecs.Map[testSliceData]
	TestNull  *ecs.Map[testNullData]
}

func (t *TestComponents) InitializeComponents(world *ecs.World) error {
	t.TestSlice = ecs.NewMap[testSliceData](world)
	t.TestNull = ecs.NewMap[testNullData](world)
	return nil
}

func TestInitComponents(t *testing.T) {
	t.Parallel()
	t.Run("正常にコンポーネントを初期化できる", func(t *testing.T) {
		t.Parallel()
		manager := ecs.NewWorld()
		gameComponents := &TestComponents{}

		components, err := InitComponents(manager, gameComponents)

		require.NoError(t, err)
		assert.NotNil(t, components)
		assert.NotNil(t, components.Game)
		assert.NotNil(t, components.Game.TestSlice)
		assert.NotNil(t, components.Game.TestNull)
	})

	t.Run("型安全性が保たれている", func(t *testing.T) {
		t.Parallel()
		manager := ecs.NewWorld()
		gameComponents := &TestComponents{}

		components, err := InitComponents(manager, gameComponents)

		require.NoError(t, err)
		// 型アサーションが不要で、直接アクセスできる
		assert.IsType(t, &TestComponents{}, components.Game)
		assert.IsType(t, &ecs.Map[testSliceData]{}, components.Game.TestSlice)
		assert.IsType(t, &ecs.Map[testNullData]{}, components.Game.TestNull)
	})
}

// FailingComponents は初期化に失敗するテスト用コンポーネント
type FailingComponents struct{}

func (f *FailingComponents) InitializeComponents(_ *ecs.World) error {
	return errors.New("初期化エラー")
}

func TestInitComponents_Error(t *testing.T) {
	t.Parallel()

	manager := ecs.NewWorld()
	components, err := InitComponents(manager, &FailingComponents{})

	require.Error(t, err)
	assert.Nil(t, components)
	assert.Contains(t, err.Error(), "初期化エラー")
}

func TestComponentInitializer(t *testing.T) {
	t.Parallel()
	t.Run("ComponentInitializerインターフェースを実装している", func(t *testing.T) {
		t.Parallel()
		var _ ComponentInitializer = &TestComponents{}
	})
}
