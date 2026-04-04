package worldhelper

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InitNewGameData は新規ゲーム開始時の初期データを設定する。
// プレイヤーが存在しない場合のみ実行される。
func InitNewGameData(world w.World) error {
	// 既にプレイヤーが存在する場合は何もしない
	hasPlayer := false
	world.Manager.Join(world.Components.Player, world.Components.FactionAlly).Visit(ecs.Visit(func(_ ecs.Entity) {
		hasPlayer = true
	}))
	if hasPlayer {
		return nil
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

	// プレイヤー生成し、デフォルトの職業を適用する
	player, err := SpawnPlayer(world, 5, 5, "Ash")
	if err != nil {
		return fmt.Errorf("プレイヤーの生成に失敗: %w", err)
	}
	professions := world.Resources.RawMaster.Raws.Professions
	if len(professions) > 0 {
		if err := ApplyProfession(world, player, professions[0]); err != nil {
			return fmt.Errorf("デフォルト職業の適用に失敗: %w", err)
		}
	}

	return nil
}
