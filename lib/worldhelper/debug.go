package worldhelper

import (
	gc "github.com/kijimaD/ruins/lib/components"
	w "github.com/kijimaD/ruins/lib/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InitDebugData はデバッグ用の初期データを設定する
// プレイヤーが存在しない場合のみ実行される
// テスト、VRT、デバッグで使用される共通のエンティティセットを生成する
func InitDebugData(world w.World) {
	// 既にプレイヤーが存在するかチェック
	{
		count := 0
		world.Manager.Join(
			world.Components.Player,
			world.Components.FactionAlly,
		).Visit(ecs.Visit(func(_ ecs.Entity) {
			count++
		}))
		// 既にプレイヤーがいる場合は何もしない
		if count > 0 {
			return
		}
	}

	// 基本アイテムの生成
	weapon1, _ := SpawnItem(world, "木刀", 1, gc.ItemLocationInBackpack)
	weapon2, _ := SpawnItem(world, "ハンドガン", 1, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "M72 LAW", 1, gc.ItemLocationInBackpack)
	armor, _ := SpawnItem(world, "西洋鎧", 1, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "作業用ヘルメット", 1, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "革のブーツ", 1, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "レイガン", 1, gc.ItemLocationInBackpack)
	// Stackableアイテム
	_, _ = SpawnItem(world, "ルビー原石", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "回復薬", 3, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "回復スプレー", 3, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "手榴弾", 5, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "パン", 10, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "鉄", 14, gc.ItemLocationInBackpack)

	// アイテム生成
	_, _ = SpawnItem(world, "木刀", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "ハンドガン", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "レイガン", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "西洋鎧", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "作業用ヘルメット", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "革のブーツ", 2, gc.ItemLocationInBackpack)

	// プレイヤー生成
	player, _ := SpawnPlayer(world, 5, 5, "セレスティン")

	// 木刀は近接武器スロットに装備
	Equip(world, weapon1, player, gc.SlotMeleeWeapon)
	// ハンドガンは遠距離武器スロットに装備
	Equip(world, weapon2, player, gc.SlotRangedWeapon)
	// 西洋鎧は胴体スロットに装備
	Equip(world, armor, player, gc.SlotTorso)

}
