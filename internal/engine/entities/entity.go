package entities

import (
	"reflect"

	"github.com/mlange-42/ark/ecs"
)

// ComponentList はエンティティ作成用のコンポーネントリスト
// Gameフィールドに EntitySpec のスライスを設定し、AddEntities でECSエンティティに変換する
type ComponentList[T any] struct {
	Entities []T // 作成するエンティティのリスト
}

// World はエンティティ作成に必要なインターフェース
// 依存性逆転のため Ark ワールドへのアクセスだけを要求する
type World interface {
	GetWorld() *ecs.World
}

// AddEntities はComponentListからECSエンティティを作成する
// EntitySpecをECSエンティティに変換し、ワールドに追加する
func AddEntities[W World, C any](world W, entityComponentList ComponentList[C]) ([]ecs.Entity, error) {
	w := world.GetWorld()
	entities := make([]ecs.Entity, len(entityComponentList.Entities))
	for iEntity := range entityComponentList.Entities {
		entity := w.NewEntity()
		if err := AddEntityComponents(w, entity, entityComponentList.Entities[iEntity]); err != nil {
			return nil, err
		}
		entities[iEntity] = entity
	}
	return entities, nil
}

// AddEntityComponents はエンティティにコンポーネントを追加する。
// EntitySpec の各ポインタフィールドを走査し、非nilなら対応するコンポーネントを付与する。
// Ark の型なしAPI（TypeID + Unsafe）を使い、コンポーネント型を実行時に解決する。
// interface フィールド（FactionType/LocationType）は具体型へ剥がして扱うため、
// struct・文字列エイリアス・interface のいずれも型名マッチ無しで統一的に処理できる。
func AddEntityComponents(w *ecs.World, entity ecs.Entity, spec any) error {
	u := w.Unsafe()
	cv := reflect.ValueOf(spec)
	for iField := range cv.NumField() {
		field := cv.Field(iField)
		if field.Kind() != reflect.Pointer || field.IsNil() {
			continue
		}
		// ポインタを剥がしてコンポーネント値を得る。interface ならさらに具体型へ剥がす
		value := field.Elem()
		if value.Kind() == reflect.Interface {
			value = value.Elem()
		}
		compType := value.Type()
		id := ecs.TypeID(w, compType)
		u.Add(entity, id)
		dst := u.Get(entity, id)
		reflect.NewAt(compType, dst).Elem().Set(value)
	}
	return nil
}
