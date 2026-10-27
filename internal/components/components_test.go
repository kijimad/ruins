package components

import (
	"math"
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

			require.Equal(t, reflect.Pointer, field.Kind(),
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

	t.Run("20を超えるフィールドをすべて初期化できる", func(t *testing.T) {
		t.Parallel()
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

			assert.Equal(t, reflect.Pointer, field.Kind(),
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

func TestOrient_Yaw(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		orient Orient
		want   float64
	}{
		{"0は北", 0, 0},
		{"1は45度", 1, math.Pi / 4},
		{"4は180度", 4, math.Pi},
		{"7は315度", 7, 7 * math.Pi / 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, tt.orient.Yaw(), 1e-9)
		})
	}
}

func TestOrient_Rotated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start Orient
		delta int
		want  Orient
	}{
		{"delta 0は変化なし", 0, 0, 0},
		{"正のdeltaは進む", 0, 3, 3},
		{"範囲を超えると巡回する", 6, 3, 1},
		{"負のdeltaは戻る", 2, -1, 1},
		{"負のdeltaで0をまたぐと末尾へ巡回する", 0, -1, 7},
		{"大きな負のdeltaでも巡回する", 0, -9, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.start.Rotated(tt.delta))
		})
	}
}

func TestCamera_Yaw(t *testing.T) {
	t.Parallel()

	// Camera.Yaw は Orient.Yaw への委譲であることを確認する
	c := Camera{Orient: 2}
	assert.InDelta(t, Orient(2).Yaw(), c.Yaw(), 1e-9)
}

func TestUpsert(t *testing.T) {
	t.Parallel()
	world := ecs.NewWorld()
	c := &Components{}
	require.NoError(t, c.InitializeComponents(world))

	e := world.NewEntity()

	// 不在なら追加する
	require.NoError(t, Upsert(world, c.Player, e, &Player{}))
	assert.True(t, c.Player.Has(e))

	// 既存なら更新する（二重追加のパニックを起こさない）
	require.NoError(t, Upsert(world, c.Player, e, &Player{}))

	// 死亡エンティティにはパニックせずエラーを返す
	world.RemoveEntity(e)
	assert.Error(t, Upsert(world, c.Player, e, &Player{}))
}
