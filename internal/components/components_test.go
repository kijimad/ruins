package components

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeComponents(t *testing.T) {
	t.Parallel()

	t.Run("正常初期化", func(t *testing.T) {
		t.Parallel()
		// Arrange
		world := ecs.NewWorld()
		components := &Components{}

		// Act
		err := components.InitializeComponents(world)

		// Assert
		require.NoError(t, err, "InitializeComponentsは成功する必要がある")

		// 全てのコンポーネントハンドル(*ecs.Map[T])が初期化されているかチェック
		val := reflect.ValueOf(components).Elem()
		typ := val.Type()

		for i := range val.NumField() {
			field := val.Field(i)
			fieldType := typ.Field(i)
			fieldName := fieldType.Name

			// Ark ではコンポーネントハンドルは全て *ecs.Map[T] ポインタになる
			require.Equal(t, reflect.Ptr, field.Kind(),
				"フィールド %s はポインタ型である必要がある", fieldName)
			assert.False(t, field.IsNil(),
				"コンポーネントハンドル %s は初期化されている必要がある", fieldName)
		}
	})

	t.Run("各コンポーネント型の初期化確認", func(t *testing.T) {
		t.Parallel()
		// Arrange
		world := ecs.NewWorld()
		components := &Components{}

		// Act
		err := components.InitializeComponents(world)

		// Assert
		require.NoError(t, err)

		// データコンポーネントのサンプルチェック
		assert.NotNil(t, components.Name, "Name ハンドルが初期化されている")
		assert.NotNil(t, components.Position, "Position ハンドルが初期化されている")
		assert.NotNil(t, components.Abilities, "Abilities ハンドルが初期化されている")

		// マーカーコンポーネントのサンプルチェック
		assert.NotNil(t, components.Player, "Player ハンドルが初期化されている")
		assert.NotNil(t, components.Dead, "Dead ハンドルが初期化されている")
	})

	t.Run("nil world でエラー", func(t *testing.T) {
		t.Parallel()
		// Arrange
		components := &Components{}

		// Act & Assert
		assert.Panics(t, func() {
			_ = components.InitializeComponents(nil)
		}, "nil worldの場合パニックが発生する")
	})

	t.Run("大量フィールドでのパフォーマンステスト", func(t *testing.T) {
		t.Parallel()
		// パフォーマンステストとして、現在のComponentsで十分な数のフィールドがある
		// Arrange
		world := ecs.NewWorld()
		components := &Components{}

		// Act
		err := components.InitializeComponents(world)

		// Assert
		require.NoError(t, err, "大量フィールドでも正常に処理される")

		// フィールド数の確認
		val := reflect.ValueOf(components).Elem()
		fieldCount := val.NumField()
		assert.Greater(t, fieldCount, 20, "十分な数のフィールドがテストされている")
	})
}

func TestComponentsStructure(t *testing.T) {
	t.Parallel()

	t.Run("全フィールドがコンポーネントハンドル型のみ", func(t *testing.T) {
		t.Parallel()
		// Components構造体の全フィールドが *ecs.Map[T] ハンドルかチェック
		val := reflect.ValueOf(&Components{}).Elem()
		typ := val.Type()

		for i := range val.NumField() {
			field := val.Field(i)
			fieldType := typ.Field(i)
			fieldName := fieldType.Name

			// Ark のコンポーネントハンドルは全て ecs.Map[T] へのポインタになる
			assert.Equal(t, reflect.Ptr, field.Kind(),
				"フィールド %s はポインタ型である必要がある", fieldName)
			assert.True(t, strings.HasPrefix(field.Type().Elem().Name(), "Map["),
				"フィールド %s の型 %v は ecs.Map ハンドルである必要がある",
				fieldName, field.Type())
		}
	})

	t.Run("公開フィールドのみ存在", func(t *testing.T) {
		t.Parallel()
		// 全てのフィールドが公開（大文字始まり）かチェック
		val := reflect.ValueOf(&Components{}).Elem()
		typ := val.Type()

		for i := range val.NumField() {
			field := val.Field(i)
			fieldType := typ.Field(i)
			fieldName := fieldType.Name

			assert.True(t, field.CanSet(),
				"フィールド %s は公開されており設定可能である必要がある", fieldName)
		}
	})
}

func TestAllAttackTypesCovered(t *testing.T) {
	t.Parallel()

	t.Run("全てのAttackTypeが正しく実装されている", func(t *testing.T) {
		t.Parallel()
		for _, at := range AllAttackTypes {
			t.Run(at.Type, func(t *testing.T) {
				t.Parallel()
				// Labelが設定されていること
				assert.NotEmpty(t, at.Label, "Labelが空である")

				// ParseAttackType()でラウンドトリップできること
				parsed, err := ParseAttackType(at.Type)
				require.NoError(t, err, "ParseAttackType()でエラーが発生した")
				assert.Equal(t, at.Type, parsed.Type)
			})
		}
	})
}
