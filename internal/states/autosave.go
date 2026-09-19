package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
)

// autoSaver は保存契機ごとにオートセーブする。契機は呼び出し側が持つ。
//
// save のテストが vrt 経由で systems と maingame を引くため、その2パッケージから save を import
// するとテスト循環になる。states は save を既に import しており循環しないので、ここに置く。
type autoSaver struct {
	manager *save.SerializationManager // テスト注入用。nil なら save が都度生成する
}

// save は現在のワールドをオートセーブする。セーブ無効や再生では何もしない。失敗はエラーで返し、扱いは
// 呼び出し側に委ねる。
func (a *autoSaver) save(world w.World) error {
	cfg := world.Resources.Config
	if !cfg.SaveLoadEnabled || cfg.DisableAutoSave {
		return nil
	}
	m := a.manager
	if m == nil {
		var err error
		m, err = save.NewSerializationManager()
		if err != nil {
			return fmt.Errorf("init save manager: %w", err)
		}
	}
	if err := m.AutoSave(world); err != nil {
		return fmt.Errorf("autosave: %w", err)
	}
	return nil
}
