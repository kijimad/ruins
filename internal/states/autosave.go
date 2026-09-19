package states

import (
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
)

// autoSaver は新規開始時と入眠時にオートセーブする。契機は呼び出し側が持つ。セーブ無効や再生では
// 何もしない。
//
// SerializationManager は保存契機の起きたときに都度生成する。契機は新規開始と入眠だけで稀なので、
// フロア入場ごとに OnStart で作るより無駄が無い。保存機構を states に置くのは import 循環回避で、
// save のテストが vrt を経由して systems と maingame を引くため、その2パッケージから save を import
// できない。states は save を既に import しており循環しない。
type autoSaver struct {
	enabled bool
	// manager はテストが一時ディレクトリの SerializationManager を注入するための穴。nil なら save が
	// 都度既定の保存先で生成する。
	manager *save.SerializationManager
}

// newAutoSaver は config を読み、オートセーブが有効かだけを決める。ここではファイル IO を伴う
// マネージャ生成はしない。
func newAutoSaver(world w.World) *autoSaver {
	cfg := world.Resources.Config
	return &autoSaver{enabled: cfg.SaveLoadEnabled && !cfg.DisableAutoSave}
}

// save は現在のワールドをオートセーブする。無効なら何もしない。マネージャは都度生成し、保存に失敗
// してもゲーム進行は止めず、ログに残すだけにする。
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
