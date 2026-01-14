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
	weapon1, _ := SpawnItem(world, "木刀", gc.ItemLocationInBackpack)
	weapon2, _ := SpawnItem(world, "ハンドガン", gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "M72 LAW", gc.ItemLocationInBackpack)
	armor, _ := SpawnItem(world, "西洋鎧", gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "作業用ヘルメット", gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "革のブーツ", gc.ItemLocationInBackpack)
	_, _ = SpawnItem(world, "レイガン", gc.ItemLocationInBackpack)
	// Stackableアイテムは数量管理
	_, _ = SpawnStackable(world, "ルビー原石", 1, gc.ItemLocationInBackpack)
	_, _ = SpawnStackable(world, "回復薬", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnStackable(world, "回復スプレー", 2, gc.ItemLocationInBackpack)
	_, _ = SpawnStackable(world, "手榴弾", 4, gc.ItemLocationInBackpack)
	_, _ = SpawnStackable(world, "鉄", 4, gc.ItemLocationInBackpack)
	_ = AddStackableCount(world, "ルビー原石", 1)
	_ = AddStackableCount(world, "回復薬", 1)
	_ = AddStackableCount(world, "回復スプレー", 1)
	_ = AddStackableCount(world, "手榴弾", 1)
	_ = AddStackableCount(world, "パン", 10)
	_ = AddStackableCount(world, "鉄", 10)

	// アイテム生成
	for i := 0; i < 2; i++ {
		_, _ = SpawnItem(world, "木刀", gc.ItemLocationInBackpack)
		_, _ = SpawnItem(world, "ハンドガン", gc.ItemLocationInBackpack)
		_, _ = SpawnItem(world, "レイガン", gc.ItemLocationInBackpack)
		_, _ = SpawnItem(world, "西洋鎧", gc.ItemLocationInBackpack)
		_, _ = SpawnItem(world, "作業用ヘルメット", gc.ItemLocationInBackpack)
		_, _ = SpawnItem(world, "革のブーツ", gc.ItemLocationInBackpack)
	}

	// プレイヤー生成
	player, _ := SpawnPlayer(world, 5, 5, "セレスティン")

	// 木刀は近接武器スロットに装備
	Equip(world, weapon1, player, gc.SlotMeleeWeapon)
	// ハンドガンは遠距離武器スロットに装備
	Equip(world, weapon2, player, gc.SlotRangedWeapon)
	// 西洋鎧は胴体スロットに装備
	Equip(world, armor, player, gc.SlotTorso)

}
