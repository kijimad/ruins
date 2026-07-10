package components

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestInitializeComponents(t *testing.T) {
	t.Parallel()

	t.Run("正常初期化", func(t *testing.T) {
		t.Parallel()
		// Arrange
		manager := ecs.NewManager()
		components := &Components{}

		// Act
		err := components.InitializeComponents(manager)

		// Assert
		require.NoError(t, err, "InitializeComponentsは成功する必要がある")

		// 全てのフィールドが初期化されているかチェック
		val := reflect.ValueOf(components).Elem()
		typ := val.Type()

		for i := range val.NumField() {
			field := val.Field(i)
			fieldName := typ.Field(i).Name

			switch field.Addr().Interface().(type) {
			case sliceComponentIniter:
				// Component[T] の埋め込み SliceComponent が初期化されているか
				embedded := field.FieldByName("SliceComponent")
				assert.False(t, embedded.IsNil(), "Component %s は初期化されている必要がある", fieldName)
			case **ecs.NullComponent:
				assert.NotNil(t, field.Interface(), "NullComponent %s は初期化されている必要がある", fieldName)
			default:
				assert.Fail(t, "未対応の型", "フィールド %s の型 %v", fieldName, field.Type())
			}
		}
	})

	t.Run("各コンポーネント型の初期化確認", func(t *testing.T) {
		t.Parallel()
		// Arrange
		manager := ecs.NewManager()
		components := &Components{}

		// Act
		err := components.InitializeComponents(manager)

		// Assert
		require.NoError(t, err)

		// Component[T]のサンプルチェック（埋め込みSliceComponentの初期化を確認）
		assert.NotNil(t, components.Name.SliceComponent, "Name コンポーネントが初期化されている")
		assert.NotNil(t, components.Position.SliceComponent, "Position コンポーネントが初期化されている")
		assert.NotNil(t, components.Abilities.SliceComponent, "Abilities コンポーネントが初期化されている")

		// NullComponentのサンプルチェック
		assert.NotNil(t, components.Player, "Player NullComponentが初期化されている")
		assert.NotNil(t, components.Dead, "Dead NullComponentが初期化されている")
	})

	t.Run("nil manager でエラー", func(t *testing.T) {
		t.Parallel()
		// Arrange
		components := &Components{}

		// Act & Assert
		assert.Panics(t, func() {
			_ = components.InitializeComponents(nil)
		}, "nil managerの場合パニックが発生する")
	})

	t.Run("未対応型はエラーを返す", func(t *testing.T) {
		t.Parallel()
		// initComponentField は Component[T]/NullComponent 以外の型でエラーを返す
		manager := ecs.NewManager()
		field := reflect.New(reflect.TypeFor[*string]()).Elem() // サポートされていない *string 型

		err := initComponentField(field, "Unsupported", manager)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported component type")
	})

	t.Run("設定不可能フィールドはエラーを返す", func(t *testing.T) {
		t.Parallel()
		// ブランク（設定不可能）フィールドに対しては not settable エラーを返す
		var s struct {
			_ *ecs.SliceComponent
		}
		manager := ecs.NewManager()
		field := reflect.ValueOf(&s).Elem().Field(0)

		err := initComponentField(field, "_", manager)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not settable")
	})

	t.Run("実際のComponentsを初期化できる", func(t *testing.T) {
		t.Parallel()
		// Arrange
		manager := ecs.NewManager()
		components := &Components{}

		// Act
		err := components.InitializeComponents(manager)

		// Assert
		require.NoError(t, err, "全フィールドが対応済み型のため正常に初期化される")

		val := reflect.ValueOf(components).Elem()
		assert.Greater(t, val.NumField(), 20, "十分な数のフィールドがテストされている")
	})
}

func TestComponentsStructure(t *testing.T) {
	t.Parallel()

	t.Run("全フィールドが対応済み型のみ", func(t *testing.T) {
		t.Parallel()
		// Components構造体の全フィールドがサポートされている型かチェック
		val := reflect.ValueOf(&Components{}).Elem()
		typ := val.Type()

		for i := range val.NumField() {
			field := val.Field(i)
			fieldName := typ.Field(i).Name

			// InitializeComponentsが扱えるのは Component[T]（sliceComponentIniter）
			// または *ecs.NullComponent のいずれか
			_, isSlice := field.Addr().Interface().(sliceComponentIniter)
			_, isNull := field.Addr().Interface().(**ecs.NullComponent)

			assert.True(t, isSlice || isNull,
				"フィールド %s の型 %v はサポートされている必要がある",
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
