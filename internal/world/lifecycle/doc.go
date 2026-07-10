// Package lifecycle はエンティティの生成・削除・配置を担う。
//
// query パッケージに依存する。
//
// # 分類ルール
//
// このパッケージに配置する関数は以下のいずれかに該当する:
//   - 実装が NewEntity / DeleteEntity / AddEntities を呼ぶ
//   - エンティティの配置（Location系コンポーネントの排他切り替え）を行う
//   - 上記の関数と密結合していて、分離すると循環依存になる
//
// # Enum Component パターン
//
// 複数のマーカー Component[T] でenum的な状態を表現する場合、
// 必ずこのパッケージのhelper関数を使用する。
// 直接Component操作すると排他制御が保証されない。
//
// # Location の使用例
//
// エンティティの位置を管理する際は、以下のhelper関数を使用する:
//
//	// ✅ 推奨: 具体的なHelper関数を使用（排他制御あり）
//	lifecycle.MoveToBackpack(world, entity, owner)
//	lifecycle.MoveToEquip(world, entity, owner, slot)
//	lifecycle.MoveToField(world, entity, &previousOwner)
//	lifecycle.MoveToField(world, entity, nil)
//
//	// ❌ 非推奨: 直接操作（排他制御なし）
//	gc.AddComponent(entity, world.Components.LocationInBackpack, &gc.LocationInBackpack{})
package lifecycle
