package save

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ComponentTypeInfo はコンポーネント型の情報
type ComponentTypeInfo struct {
	Name           string                                                         // 型名
	Type           reflect.Type                                                   // 型情報
	FieldName      string                                                         // Componentsフィールド名(例: "Name", "Pools")
	ComponentRef   interface{}                                                    // ECSコンポーネントへの参照
	ExtractFunc    func(w.World, ecs.Entity) (interface{}, bool)                  // エンティティからコンポーネントを抽出
	RestoreFunc    func(w.World, ecs.Entity, interface{}) error                   // エンティティにコンポーネントを復元
	ResolveRefFunc func(w.World, ecs.Entity, interface{}, *StableIDManager) error // エンティティ参照を解決
}

// ComponentRegistry はコンポーネント型の自動検出と管理を行う
type ComponentRegistry struct {
	types       map[reflect.Type]*ComponentTypeInfo
	nameToType  map[string]reflect.Type
	mutex       sync.RWMutex
	initialized bool
}

// NewComponentRegistry は新しいComponentRegistryを作成
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		types:      make(map[reflect.Type]*ComponentTypeInfo),
		nameToType: make(map[string]reflect.Type),
	}
}

// getComponentTypeMap はComponentsフィールド名からコンポーネント型へのマッピングを返す
// 新しいコンポーネントを保存対象にする場合は、ここに追加する
func getComponentTypeMap() map[string]reflect.Type {
	return map[string]reflect.Type{
		// プレイヤー・味方
		"Player":      reflect.TypeOf(&gc.Player{}),
		"FactionAlly": reflect.TypeOf(&gc.FactionAllyData{}),

		// アイテム
		"Item":                         reflect.TypeOf(&gc.Item{}),
		"ItemLocationInPlayerBackpack": reflect.TypeOf(&gc.LocationInPlayerBackpack{}),
		"ItemLocationEquipped":         reflect.TypeOf(&gc.LocationEquipped{}),

		// イベント
		"EquipmentChanged": reflect.TypeOf(&gc.EquipmentChanged{}),

		// データ
		"Name":        reflect.TypeOf(&gc.Name{}),
		"Description": reflect.TypeOf(&gc.Description{}),
		"Pools":       reflect.TypeOf(&gc.Pools{}),
		"TurnBased":   reflect.TypeOf(&gc.TurnBased{}),
		"Attributes":  reflect.TypeOf(&gc.Attributes{}),

		// 表示
		"SpriteRender": reflect.TypeOf(&gc.SpriteRender{}),
		"LightSource":  reflect.TypeOf(&gc.LightSource{}),

		// アイテム属性
		"Wearable":  reflect.TypeOf(&gc.Wearable{}),
		"Weapon":    reflect.TypeOf(&gc.Weapon{}),
		"Stackable": reflect.TypeOf(&gc.Stackable{}),
		"Value":     reflect.TypeOf(&gc.Value{}),
		"Attack":    reflect.TypeOf(&gc.Attack{}),
		"Recipe":    reflect.TypeOf(&gc.Recipe{}),

		// アイテム効果
		"Consumable":        reflect.TypeOf(&gc.Consumable{}),
		"ProvidesHealing":   reflect.TypeOf(&gc.ProvidesHealing{}),
		"ProvidesNutrition": reflect.TypeOf(&gc.ProvidesNutrition{}),
		"InflictsDamage":    reflect.TypeOf(&gc.InflictsDamage{}),

		// その他
		"Wallet": reflect.TypeOf(&gc.Wallet{}),
	}
}

// InitializeFromWorld はワールドから自動的にコンポーネント型を検出・登録
// save:"true"タグが付いたComponentsフィールドを自動的に登録する
func (r *ComponentRegistry) InitializeFromWorld(world w.World) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // 既に初期化済み
	}

	components := world.Components

	// Componentsのフィールドをリフレクションでスキャン
	componentsValue := reflect.ValueOf(components).Elem()
	componentsType := componentsValue.Type()

	// 特別処理が必要なコンポーネントのマップ
	// インターフェースやエンティティ参照を含むコンポーネントのみ個別処理
	specialExtractFuncs := map[string]func(w.World, ecs.Entity) (interface{}, bool){
		"ItemLocationEquipped": r.extractItemLocationEquipped,
		"ProvidesHealing":      r.extractProvidesHealing,
	}

	specialRestoreFuncs := map[string]func(w.World, ecs.Entity, interface{}) error{
		"ItemLocationEquipped": r.restoreItemLocationEquipped,
		"ProvidesHealing":      r.restoreProvidesHealing,
	}

	// LocationEquippedは特別処理（エンティティ参照解決が必要）
	specialResolveFuncs := map[string]func(w.World, ecs.Entity, interface{}, *StableIDManager) error{
		"ItemLocationEquipped": r.resolveLocationEquippedRefs,
	}

	// フィールド名→コンポーネント型のマッピング
	// save:"true"タグと組み合わせて使用
	componentTypeMap := getComponentTypeMap()

	for i := 0; i < componentsType.NumField(); i++ {
		field := componentsType.Field(i)

		// save:"true"タグがあるフィールドのみ処理
		if field.Tag.Get("save") != "true" {
			continue
		}

		componentName := field.Name
		componentRef := componentsValue.Field(i).Interface()

		// フィールド名から型を取得
		componentType, exists := componentTypeMap[componentName]
		if !exists {
			// 未知のコンポーネント型
			continue
		}

		// NullComponentかどうかを判定
		_, isNull := componentRef.(*ecs.NullComponent)

		if isNull {
			r.registerNullComponent(componentType, componentName, componentRef)
		} else {
			// extract/restore関数を取得
			// 特別処理が必要なコンポーネントのみspecialFuncsから取得、それ以外は汎用関数を使用
			extractFunc := specialExtractFuncs[componentName]
			restoreFunc := specialRestoreFuncs[componentName]
			resolveFunc := specialResolveFuncs[componentName]

			if extractFunc != nil && restoreFunc != nil {
				// 特別処理が必要なコンポーネント
				r.registerComponent(componentType, componentName, componentRef, extractFunc, restoreFunc, resolveFunc)
			} else {
				// JSONタグを使った汎用処理
				elemType := componentType.Elem()
				genericExtract := r.createGenericExtract(componentRef)
				genericRestore := r.createGenericRestore(componentName, elemType)
				r.registerComponent(componentType, componentName, componentRef, genericExtract, genericRestore, nil)
			}
		}
	}

	r.initialized = true
	return nil
}

// registerComponent は単一コンポーネント型を登録
func (r *ComponentRegistry) registerComponent(
	typ reflect.Type,
	fieldName string,
	componentRef interface{},
	extractFunc func(w.World, ecs.Entity) (interface{}, bool),
	restoreFunc func(w.World, ecs.Entity, interface{}) error,
	resolveRefFunc func(w.World, ecs.Entity, interface{}, *StableIDManager) error,
) {
	// ポインタ型から要素型を取得
	elemType := typ.Elem()

	info := &ComponentTypeInfo{
		Name:           elemType.Name(),
		Type:           elemType,
		FieldName:      fieldName,
		ComponentRef:   componentRef,
		ExtractFunc:    extractFunc,
		RestoreFunc:    restoreFunc,
		ResolveRefFunc: resolveRefFunc,
	}

	r.types[elemType] = info
	r.nameToType[elemType.Name()] = elemType
}

// registerNullComponent はNullComponent型を登録
func (r *ComponentRegistry) registerNullComponent(typ reflect.Type, fieldName string, componentRef interface{}) {
	elemType := typ.Elem()

	info := &ComponentTypeInfo{
		Name:         elemType.Name(),
		Type:         elemType,
		FieldName:    fieldName,
		ComponentRef: componentRef,
		ExtractFunc: func(world w.World, entity ecs.Entity) (interface{}, bool) {
			// NullComponentの存在チェック
			switch elemType.Name() {
			case "Player":
				return struct{}{}, entity.HasComponent(world.Components.Player)
			case "FactionAllyData":
				return struct{}{}, entity.HasComponent(world.Components.FactionAlly)
			case "Item":
				return struct{}{}, entity.HasComponent(world.Components.Item)
			case "LocationInPlayerBackpack":
				return struct{}{}, entity.HasComponent(world.Components.ItemLocationInPlayerBackpack)
			case "EquipmentChanged":
				return struct{}{}, entity.HasComponent(world.Components.EquipmentChanged)
			}
			return nil, false
		},
		RestoreFunc: func(world w.World, entity ecs.Entity, _ interface{}) error {
			// NullComponentを追加
			switch elemType.Name() {
			case "Player":
				entity.AddComponent(world.Components.Player, &gc.Player{})
			case "FactionAllyData":
				entity.AddComponent(world.Components.FactionAlly, &gc.FactionAllyData{})
			case "Item":
				entity.AddComponent(world.Components.Item, &gc.Item{})
			case "LocationInPlayerBackpack":
				entity.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})
			case "EquipmentChanged":
				entity.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
			default:
				return fmt.Errorf("unknown NullComponent type: %s", elemType.Name())
			}
			return nil
		},
		ResolveRefFunc: nil, // NullComponentには参照解決は不要
	}

	r.types[elemType] = info
	r.nameToType[elemType.Name()] = elemType
}

// GetTypeInfo は型情報を取得
func (r *ComponentRegistry) GetTypeInfo(typ reflect.Type) (*ComponentTypeInfo, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	info, exists := r.types[typ]
	return info, exists
}

// GetTypeInfoByName は名前から型情報を取得
func (r *ComponentRegistry) GetTypeInfoByName(name string) (*ComponentTypeInfo, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	typ, exists := r.nameToType[name]
	if !exists {
		return nil, false
	}

	info, exists := r.types[typ]
	return info, exists
}

// GetAllTypes は登録されている全ての型を取得（名前順でソート済み）
func (r *ComponentRegistry) GetAllTypes() []*ComponentTypeInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	types := make([]*ComponentTypeInfo, 0, len(r.types))
	for _, info := range r.types {
		types = append(types, info)
	}

	// 名前順でソートして決定的な順序にする
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	return types
}

// createGenericExtract は汎用的なextract関数を生成(SliceComponent用)
// JSONタグを使った自動シリアライズに対応
func (r *ComponentRegistry) createGenericExtract(componentRef interface{}) func(w.World, ecs.Entity) (interface{}, bool) {
	// componentRefを*ecs.SliceComponentにキャスト
	sliceComp, ok := componentRef.(*ecs.SliceComponent)
	if !ok {
		// SliceComponentでない場合はnilを返す
		return func(_ w.World, _ ecs.Entity) (interface{}, bool) {
			return nil, false
		}
	}

	return func(_ w.World, entity ecs.Entity) (interface{}, bool) {
		if !entity.HasComponent(sliceComp) {
			return nil, false
		}

		dataPtr := sliceComp.Get(entity)

		// ポインタをデリファレンス
		data := reflect.ValueOf(dataPtr).Elem().Interface()

		return data, true
	}
}

// createGenericRestore は汎用的なrestore関数を生成(SliceComponent用)
// JSONタグを使った自動デシリアライズに対応
func (r *ComponentRegistry) createGenericRestore(fieldName string, _ reflect.Type) func(w.World, ecs.Entity, interface{}) error {
	return func(world w.World, entity ecs.Entity, data interface{}) error {
		// worldからComponentsフィールドを取得
		componentsValue := reflect.ValueOf(world.Components).Elem()
		componentsType := componentsValue.Type()

		// フィールド名からコンポーネントを取得
		field, found := componentsType.FieldByName(fieldName)
		if !found {
			return fmt.Errorf("field %s not found in Components", fieldName)
		}

		// フィールドの値を取得
		fieldValue := componentsValue.FieldByName(field.Name)
		componentRef := fieldValue.Interface()

		// SliceComponentにキャスト
		sliceComp, ok := componentRef.(*ecs.SliceComponent)
		if !ok {
			return fmt.Errorf("field %s is not a SliceComponent", fieldName)
		}

		// dataを適切な型にキャストしてポインタ化
		dataValue := reflect.ValueOf(data)

		// 実際の型を使用してポインタを作成
		actualType := dataValue.Type()

		// ポインタ化
		dataPtr := reflect.New(actualType)
		dataPtr.Elem().Set(dataValue)

		// AddComponent
		entity.AddComponent(sliceComp, dataPtr.Interface())

		return nil
	}
}

// ItemLocationEquipped コンポーネントの処理
func (r *ComponentRegistry) extractItemLocationEquipped(world w.World, entity ecs.Entity) (interface{}, bool) {
	if !entity.HasComponent(world.Components.ItemLocationEquipped) {
		return nil, false
	}
	equipped := world.Components.ItemLocationEquipped.Get(entity).(*gc.LocationEquipped)
	return *equipped, true
}

func (r *ComponentRegistry) restoreItemLocationEquipped(world w.World, entity ecs.Entity, data interface{}) error {
	equipped, ok := data.(gc.LocationEquipped)
	if !ok {
		return fmt.Errorf("invalid LocationEquipped data type: %T", data)
	}
	entity.AddComponent(world.Components.ItemLocationEquipped, &equipped)
	return nil
}

func (r *ComponentRegistry) resolveLocationEquippedRefs(_ w.World, _ ecs.Entity, _ interface{}, _ *StableIDManager) error {
	// エンティティ参照の解決はSerializationManagerで実装
	return nil
}

// ProvidesHealing コンポーネントの処理
func (r *ComponentRegistry) extractProvidesHealing(world w.World, entity ecs.Entity) (interface{}, bool) {
	if !entity.HasComponent(world.Components.ProvidesHealing) {
		return nil, false
	}
	healing := world.Components.ProvidesHealing.Get(entity).(*gc.ProvidesHealing)

	// Amounterインターフェースを具体的な型に変換してシリアライズ
	var amountData map[string]interface{}
	switch a := healing.Amount.(type) {
	case gc.RatioAmount:
		amountData = map[string]interface{}{
			"type":  "ratio",
			"ratio": a.Ratio,
		}
	case gc.NumeralAmount:
		amountData = map[string]interface{}{
			"type":    "numeral",
			"numeral": a.Numeral,
		}
	default:
		// デフォルトでRatio 0.5を使用
		amountData = map[string]interface{}{
			"type":  "ratio",
			"ratio": 0.5,
		}
	}

	return map[string]interface{}{
		"amount": amountData,
	}, true
}

func (r *ComponentRegistry) restoreProvidesHealing(world w.World, entity ecs.Entity, data interface{}) error {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid ProvidesHealing data type: %T", data)
	}

	amountData, ok := dataMap["amount"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid ProvidesHealing amount data")
	}

	var amount gc.Amounter
	amountType, _ := amountData["type"].(string)
	switch amountType {
	case "ratio":
		if ratio, ok := amountData["ratio"].(float64); ok {
			amount = gc.RatioAmount{Ratio: ratio}
		}
	case "numeral":
		if numeral, ok := amountData["numeral"].(float64); ok {
			amount = gc.NumeralAmount{Numeral: int(numeral)}
		}
	default:
		return fmt.Errorf("unknown amount type: %s", amountType)
	}

	healing := &gc.ProvidesHealing{
		Amount: amount,
	}
	entity.AddComponent(world.Components.ProvidesHealing, healing)
	return nil
}
