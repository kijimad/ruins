package worldhelper

import (
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InitNewGameData は新規ゲーム開始時の初期データを設定する
// プレイヤーが存在しない場合のみ実行される
func InitNewGameData(world w.World) {
	// 既にプレイヤーが存在する場合は何もしない
	hasPlayer := false
	world.Manager.Join(world.Components.Player, world.Components.FactionAlly).Visit(ecs.Visit(func(_ ecs.Entity) {
		hasPlayer = true
	}))
	if hasPlayer {
		return
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

	// プレイヤー生成
	_, _ = SpawnPlayer(world, 5, 5, "Ash")
}
