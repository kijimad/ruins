package entities

import (
	"fmt"
	"reflect"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// ComponentList はエンティティ作成用のコンポーネントリスト
// Gameフィールドに EntitySpec のスライスを設定し、AddEntities でECSエンティティに変換する
type ComponentList[T any] struct {
	Entities []T // 作成するエンティティのリスト
}

// World represents the required interface for entity creation
// 依存性逆転のためのメソッドを定義する
type World interface {
	GetManager() *ecs.Manager
	GetComponents() interface{}
}

// AddEntities はComponentListからECSエンティティを作成する
// EntitySpecをECSエンティティに変換し、ワールドに追加する
func AddEntities[W World, C any](world W, entityComponentList ComponentList[C]) ([]ecs.Entity, error) {
	// Create new entities and add engine components
	entities := make([]ecs.Entity, len(entityComponentList.Entities))
	for iEntity := range entityComponentList.Entities {
		entities[iEntity] = world.GetManager().NewEntity()
		if err := AddEntityComponents(entities[iEntity], world.GetComponents(), entityComponentList.Entities[iEntity]); err != nil {
			return nil, err
		}
	}

	// Add game components
	if entityComponentList.Entities != nil {
		if len(entityComponentList.Entities) != len(entities) {
			return nil, fmt.Errorf("incorrect size for game component list")
		}
		for iEntity := range entities {
			if err := AddEntityComponents(entities[iEntity], world.GetComponents(), entityComponentList.Entities[iEntity]); err != nil {
				return nil, err
			}
		}
	}
	return entities, nil
}

// AddEntityComponents はエンティティにコンポーネントを追加する
// EntitySpec の各フィールドを対応する ECS コンポーネントに変換して追加する
func AddEntityComponents(entity ecs.Entity, ecsComponentList interface{}, components interface{}) error {
	// 追加先のコンポーネントリスト。コンポーネントのスライス群
	ecv := reflect.ValueOf(ecsComponentList).Elem()
	// 追加するコンポーネント
	cv := reflect.ValueOf(components)
	for iField := 0; iField < cv.NumField(); iField++ {
		if !cv.Field(iField).IsNil() {
			component := cv.Field(iField).Elem()
			value := reflect.New(reflect.TypeOf(component.Interface()))

			switch component.Kind() {
			case reflect.Struct:
				// 追加対象コンポーネントの型名を使って、追加先コンポーネントのフィールドを対応付けて値を設定する
				value.Elem().Set(component)
				ecsComponent := ecv.FieldByName(component.Type().Name()).Interface().(ecs.DataComponent)
				entity.AddComponent(ecsComponent, value.Interface())
			case reflect.Interface:
				// Stringer インターフェースだけ対応している。Componentsに対応するフィールド名が必須なため
				if component.Type().Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem()) {
					method := component.MethodByName("String")
					if !method.IsValid() {
						return fmt.Errorf("String() に失敗した")
					}
					results := method.Call(nil)
					if len(results) != 1 {
						return fmt.Errorf("String() の返り値の取得に失敗した")
					}
					v := component.Elem().Interface()
					value.Elem().Set(reflect.ValueOf(v))

					result := results[0].Interface().(string)
					ecsComponent := ecv.FieldByName(result).Interface().(ecs.DataComponent)
					entity.AddComponent(ecsComponent, value.Interface())
				}
			case reflect.String:
				// 文字列ベースの型エイリアス（PropType等）を処理
				// 型名を使ってコンポーネントを特定する
				value.Elem().Set(component)
				ecsComponent := ecv.FieldByName(component.Type().Name()).Interface().(ecs.DataComponent)
				entity.AddComponent(ecsComponent, value.Interface())
			default:
				return fmt.Errorf("EntitySpecフィールドに指定された型の処理は定義されていない: %s", component.Kind())
			}
		}
	}
	return nil
}
