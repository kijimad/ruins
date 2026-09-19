package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
)

// autoSave は現在のワールドをオートセーブする。セーブ無効や再生では何もしない。manager が nil なら
// 既定の保存先で都度生成する。テストは一時ディレクトリの manager を渡して隔離する。失敗はエラーで返し、
// 扱いは呼び出し側に委ねる。
//
// テストが vrt 経由で systems と maingame を引くため、その2パッケージから save を import すると
// テスト循環になる。states は save を既に import しており循環しないので、ここに置く。
func autoSave(world w.World, manager *save.SerializationManager) error {
	cfg := world.Resources.Config
	if !cfg.SaveLoadEnabled || cfg.DisableAutoSave {
		return nil
	}
	m := manager
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
