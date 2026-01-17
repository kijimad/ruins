// Package worldhelper はワールド操作の高レベルAPIを提供します。
//
// # Enum Component パターン
//
// 複数のNullComponent/SliceComponentでenum的な状態を表現する場合、
// 必ずこのパッケージのhelper関数を使用してください。
// 直接Component操作すると排他制御が保証されません。
//
// # ItemLocation の使用例
//
// アイテムの位置を管理する際は、以下のhelper関数を使用します：
//
//	// ✅ 推奨: 具体的なHelper関数を使用（排他制御あり）
//	worldhelper.MoveToBackpack(world, item, owner)            // バックパックに移動（EquipmentChanged, InventoryChangedフラグ付き）
//	worldhelper.MoveToEquip(world, item, owner, slot)         // 装備（EquipmentChanged, InventoryChangedフラグ付き）
//	worldhelper.MoveToField(world, item, owner)               // フィールドにドロップ（InventoryChangedフラグ付き）
//
//	// ❌ 非推奨: 直接操作（排他制御なし）
//	item.AddComponent(world.Components.ItemLocationInPlayerBackpack)
//
// 位置の判定は型取得してswitch分岐、またはHasComponentで判定します：
//
//	// ✅ HasComponentで直接判定
//	if item.HasComponent(world.Components.ItemLocationInPlayerBackpack) {
//	    // バックパック内処理
//	}
//
//	// ✅ 型取得してswitch分岐
//	locType, ok := worldhelper.GetItemLocationType(world, item)
//	if ok {
//	    switch locType {
//	    case worldhelper.ItemLocationInPlayerBackpack:
//	        // バックパック処理
//	    case worldhelper.ItemLocationEquipped:
//	        equipped := world.Components.ItemLocationEquipped.Get(item).(*gc.LocationEquipped)
//	        // 装備処理（equipped.Owner, equipped.EquipmentSlotにアクセス可能）
//	    case worldhelper.ItemLocationOnField:
//	        // フィールド処理
//	    }
//	}
//
// # Faction の使用例
//
// エンティティの派閥を管理する際も、helper関数を使用します：
//
//	// ✅ 推奨: 具体的なHelper関数を使用（排他制御あり）
//	worldhelper.MakeAlly(world, entity)
//	worldhelper.MakeEnemy(world, entity)
//	worldhelper.MakeNeutral(world, entity)
//
//	// ❌ 非推奨: 直接操作（排他制御なし）
//	entity.AddComponent(world.Components.FactionAlly, &gc.FactionAllyData{})
//
// 派閥の判定は型取得してswitch分岐、またはHasComponentで判定します：
//
//	// ✅ HasComponentで直接判定
//	if entity.HasComponent(world.Components.FactionAlly) {
//	    // 味方処理
//	}
//
//	// ✅ 型取得してswitch分岐
//	factionType, ok := worldhelper.GetFactionType(world, entity)
//	if ok {
//	    switch factionType {
//	    case worldhelper.FactionAlly:
//	        // 味方処理
//	    case worldhelper.FactionEnemy:
//	        // 敵処理
//	    case worldhelper.FactionNeutral:
//	        // 中立処理
//	    }
//	}
//
// # 設計原則
//
// このパターンはECSのセオリーに従っています：
//   - コンポーネント: 単純なデータ構造（状態はコンポーネントの有無で表現）
//   - System/Helper: ロジックと排他制御を担当
//
// Helper関数を使用することで：
//   - 排他制御を保証（複数の状態が同時に存在しない）
//   - 意図が明確（関数名で何をしているか分かる）
//   - 型安全（間違った操作をコンパイル時に検出）
package worldhelper
