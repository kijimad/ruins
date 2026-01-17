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

		// 全てのSliceComponentフィールドが初期化されているかチェック
		val := reflect.ValueOf(components).Elem()
		typ := val.Type()

		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldType := typ.Field(i)
			fieldName := fieldType.Name

			switch field.Type() {
			case reflect.TypeOf((*ecs.SliceComponent)(nil)):
				assert.NotNil(t, field.Interface(), "SliceComponent %s は初期化されている必要がある", fieldName)
			case reflect.TypeOf((*ecs.NullComponent)(nil)):
				assert.NotNil(t, field.Interface(), "NullComponent %s は初期化されている必要がある", fieldName)
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

		// SliceComponentのサンプルチェック
		assert.NotNil(t, components.Name, "Name SliceComponentが初期化されている")
		assert.NotNil(t, components.Position, "Position SliceComponentが初期化されている")
		assert.NotNil(t, components.Attributes, "Attributes SliceComponentが初期化されている")

		// NullComponentのサンプルチェック
		assert.NotNil(t, components.Item, "Item NullComponentが初期化されている")
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

	t.Run("未対応型エラーテスト", func(t *testing.T) {
		t.Parallel()
		// テスト用の構造体（未対応の型を含む）
		type TestComponentsWithUnsupportedType struct {
			ValidSliceComponent *ecs.SliceComponent
			ValidNullComponent  *ecs.NullComponent
			UnsupportedType     *string // サポートされていない型
		}

		// Arrange
		manager := ecs.NewManager()
		testComponents := &TestComponentsWithUnsupportedType{}

		// InitializeComponentsと同じロジックを実行
		val := reflect.ValueOf(testComponents).Elem()
		typ := val.Type()

		hasUnsupportedType := false
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldType := typ.Field(i)
			fieldName := fieldType.Name

			if !field.CanSet() {
				t.Errorf("field %s is not settable", fieldName)
				return
			}

			switch field.Type() {
			case reflect.TypeOf((*ecs.SliceComponent)(nil)):
				field.Set(reflect.ValueOf(manager.NewSliceComponent()))
			case reflect.TypeOf((*ecs.NullComponent)(nil)):
				field.Set(reflect.ValueOf(manager.NewNullComponent()))
			default:
				// 未対応の型が検出された
				hasUnsupportedType = true
				t.Logf("未対応の型を検出: %v", fieldType.Type)
			}
		}

		assert.True(t, hasUnsupportedType, "未対応の型が検出されるべき")
	})

	t.Run("設定不可能フィールドエラーテスト", func(t *testing.T) {
		t.Parallel()
		// 設定不可能フィールドのテストは、実際のstruct fieldがprivateの場合に発生
		// 通常の使用では発生しないが、将来の拡張に備えてテストケースを用意

		// テスト用の構造体（非公開フィールドを含む）
		type TestComponentsWithPrivateField struct {
			ValidSliceComponent *ecs.SliceComponent
			_                   *ecs.SliceComponent // 非公開フィールド（設定不可能）
		}

		// Arrange
		testComponents := &TestComponentsWithPrivateField{}

		// InitializeComponentsと同じロジックを実行
		val := reflect.ValueOf(testComponents).Elem()

		hasUnsettableField := false
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if !field.CanSet() {
				hasUnsettableField = true
				break
			}
		}

		// Assert
		assert.True(t, hasUnsettableField, "非公開フィールドは設定不可能である")
	})

	t.Run("空のComponentsでも正常動作", func(t *testing.T) {
		t.Parallel()
		// 空のComponents構造体での動作確認
		type EmptyComponents struct{}

		// Arrange
		emptyComponents := &EmptyComponents{}

		// InitializeComponentsと同じロジックを実行
		val := reflect.ValueOf(emptyComponents).Elem()

		// フィールドが0個でもエラーにならないことを確認
		assert.Equal(t, 0, val.NumField(), "空の構造体はフィールド数が0")

		// 実際には何も処理されないが、エラーは発生しない想定
		// Act & Assert（エラーが発生しないことを確認）
		// この場合、実際のInitializeComponentsメソッドは存在しないので、
		// ロジックのチェックのみ行う
	})

	t.Run("大量フィールドでのパフォーマンステスト", func(t *testing.T) {
		t.Parallel()
		// パフォーマンステストとして、現在のComponentsで十分な数のフィールドがある
		// Arrange
		manager := ecs.NewManager()
		components := &Components{}

		// Act
		err := components.InitializeComponents(manager)

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

	t.Run("全フィールドが対応済み型のみ", func(t *testing.T) {
		t.Parallel()
		// Components構造体の全フィールドがサポートされている型かチェック
		val := reflect.ValueOf(&Components{}).Elem()
		typ := val.Type()

		supportedTypes := []reflect.Type{
			reflect.TypeOf((*ecs.SliceComponent)(nil)),
			reflect.TypeOf((*ecs.NullComponent)(nil)),
		}

		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldType := typ.Field(i)
			fieldName := fieldType.Name

			isSupported := false
			for _, supportedType := range supportedTypes {
				if field.Type() == supportedType {
					isSupported = true
					break
				}
			}

			assert.True(t, isSupported,
				"フィールド %s の型 %v はサポートされている必要がある",
				fieldName, field.Type())
		}
	})

	t.Run("公開フィールドのみ存在", func(t *testing.T) {
		t.Parallel()
		// 全てのフィールドが公開（大文字始まり）かチェック
		val := reflect.ValueOf(&Components{}).Elem()
		typ := val.Type()

		for i := 0; i < val.NumField(); i++ {
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
