package states

import (
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
)

// autoSaver は新規開始時と入眠時にオートセーブする。SerializationManager を1つ所有し、セーブ無効や
// 生成失敗、再生のときは何もしない。DungeonState が契機の起きた箇所から save を呼ぶ。
//
// SerializationManager を保存機構の外から使うため systems でなく states に置く。save.test は vrt を
// 経由して systems と maingame を引くので、その2パッケージから save を import するとテストで import
// 循環になる。states は save を既に import しており循環しない。
type autoSaver struct {
	manager *save.SerializationManager
}

// newAutoSaver は autoSaver を初期化する。セーブ無効時や再生時、マネージャ生成失敗時は manager を nil
// のままにし、save を no-op にする。
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

// save は現在のワールドをオートセーブする。契機は呼び出し側が持つ。保存に失敗してもゲーム進行は
// 止めず、ログに残すだけにする。
func (a *autoSaver) save(world w.World) {
	if a.manager == nil {
		return
	}
	if err := a.manager.AutoSave(world); err != nil {
		logger.New(logger.CategorySave).Warn("autosave failed", "error", err.Error())
	}
}
