package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
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

	// 新しいゲーム開始時のみゲームログをクリア
	gamelog.FieldLog.Clear()
	gamelog.SceneLog.Clear()
	// 操作ガイドを表示
	gamelog.New(gamelog.FieldLog).
		System("WASD: 移動する。").
		Log()
	gamelog.New(gamelog.FieldLog).
		System("Mキー: 拠点メニューを開く。").
		Log()
	gamelog.New(gamelog.FieldLog).
		System("Spaceキー: アクションメニューを開く。").
		Log()

	// 基本アイテムの生成
	weapon1, _ := SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
	weapon2, _ := SpawnItem(world, "ハンドガン", 1, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "M72 LAW", 1, gc.ItemLocationInPlayerBackpack)
	armor, _ := SpawnItem(world, "西洋鎧", 1, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "作業用ヘルメット", 1, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "革のブーツ", 1, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "レイガン", 1, gc.ItemLocationInPlayerBackpack)
	// Stackableアイテム
	_, _ = SpawnItem(world, "ルビー原石", 2, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "回復薬", 3, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "手榴弾", 5, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "パン", 10, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "鉄", 14, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "コーラ", 2, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "緑ハーブ", 2, gc.ItemLocationInPlayerBackpack)

	// アイテム生成
	_, _ = SpawnItem(world, "木刀", 2, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "ハンドガン", 2, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "レイガン", 2, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "西洋鎧", 2, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "作業用ヘルメット", 2, gc.ItemLocationInPlayerBackpack)
	_, _ = SpawnItem(world, "革のブーツ", 2, gc.ItemLocationInPlayerBackpack)

	// プレイヤー生成
	player, _ := SpawnPlayer(world, 5, 5, "セレスティン")

	// 木刀は武器スロット1に装備
	MoveToEquip(world, weapon1, player, gc.SlotWeapon1)
	// ハンドガンは武器スロット2に装備
	MoveToEquip(world, weapon2, player, gc.SlotWeapon2)
	// 西洋鎧は胴体スロットに装備
	MoveToEquip(world, armor, player, gc.SlotTorso)
}
