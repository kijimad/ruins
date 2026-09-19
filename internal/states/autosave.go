package states

import (
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
)

// autoSaver は保存契機ごとにオートセーブする。契機は呼び出し側が持つ。
//
// save のテストが vrt 経由で systems と maingame を引くため、その2パッケージから save を import
// するとテスト循環になる。states は save を既に import しており循環しないので、ここに置く。
type autoSaver struct {
	enabled bool
	manager *save.SerializationManager // テスト注入用。nil なら save が都度生成する
}

func newAutoSaver(world w.World) *autoSaver {
	cfg := world.Resources.Config
	return &autoSaver{enabled: cfg.SaveLoadEnabled && !cfg.DisableAutoSave}
}

// save は現在のワールドをオートセーブする。失敗してもゲーム進行は止めず、ログに残すだけにする。
func (a *autoSaver) save(world w.World) {
	if !a.enabled {
		return
	}
	m := a.manager
	if m == nil {
		var err error
		m, err = save.NewSerializationManager()
		if err != nil {
			logger.New(logger.CategorySave).Warn("autosave: failed to init manager", "error", err.Error())
			return
		}
	}
	if err := m.AutoSave(world); err != nil {
		logger.New(logger.CategorySave).Warn("autosave failed", "error", err.Error())
	}
}
