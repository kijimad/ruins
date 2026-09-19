package states

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// autoSaveInterval はオートセーブの間隔。1日は gc.TurnsPerDay ターンなので約0.1日ごと。
const autoSaveInterval consts.Turn = 150

// autoSaver は一定ターンごとと新規開始直後にオートセーブする。SerializationManager を1つ所有し、
// セーブ無効や生成失敗のときは何もしない。DungeonState が毎フレーム maybeSave を呼ぶ。
//
// SerializationManager を保存機構の外から使うため systems でなく states に置く。save.test は
// vrt を経由して systems と maingame を引くので、その2パッケージから save を import すると
// テストで import 循環になる。states は save を既に import しており循環しない。
type autoSaver struct {
	manager *save.SerializationManager
}

// newAutoSaver は autoSaver を初期化する。セーブ無効時やマネージャ生成失敗時は manager を nil の
// ままにし、maybeSave を no-op にする。
func newAutoSaver(world w.World) *autoSaver {
	if !world.Resources.Config.SaveLoadEnabled || world.Resources.Config.DisableAutoSave {
		return &autoSaver{}
	}
	m, err := save.NewSerializationManager()
	if err != nil {
		logger.New(logger.CategorySave).Warn("autosave: failed to init manager", "error", err.Error())
		return &autoSaver{}
	}
	return &autoSaver{manager: m}
}

// maybeSave は発火条件を満たすときオートセーブする。継続アクティビティ中は抑止し、前回保存から
// autoSaveInterval ターン経過で保存する。LastAutoSaveTurn が 0 の未保存状態は新規開始直後の初回
// 保存として扱う。保存に失敗してもゲーム進行は止めない。
func (a *autoSaver) maybeSave(world w.World) {
	if a.manager == nil {
		return
	}
	if playerHasActivity(world) {
		return
	}
	ts := query.GetTurnState(world)
	if ts.LastAutoSaveTurn != 0 && ts.TurnNumber-ts.LastAutoSaveTurn < autoSaveInterval {
		return
	}
	if err := a.manager.AutoSave(world); err != nil {
		logger.New(logger.CategorySave).Warn("autosave failed", "error", err.Error())
		return
	}
	ts.LastAutoSaveTurn = ts.TurnNumber
}

// playerHasActivity はプレイヤーが継続アクティビティ中かを返す。
func playerHasActivity(world w.World) bool {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return false
	}
	return query.HasActivity(world, playerEntity)
}
