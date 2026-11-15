package systems

import (
	"github.com/kijimaD/ruins/internal/overworld"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// AreaEntrySystem は地上でプレイヤーのいるエリアが変わったら、エリアの名前をスプラッシュで出す。
// 荒れ地に出たときは出さない。直前のエリアはセーブしないので、最初に見たエリアは記憶するだけにし、
// 開始とロードの直後には出さない
type AreaEntrySystem struct {
	last     string
	observed bool
}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys AreaEntrySystem) String() string {
	return "AreaEntrySystem"
}

// Update はプレイヤーのエリアを前回と比べ、別のエリアに入ったらスプラッシュを出す
// w.Updater interfaceを実装
func (sys *AreaEntrySystem) Update(world w.World) error {
	area, ok := overworld.PlayerArea(world)
	if !ok {
		return nil
	}
	if sys.observe(area) {
		lifecycle.SpawnSplashText(world, query.T(world, area.Name))
	}
	return nil
}

// observe は現在のエリアを記憶し、前回と別のエリアに入ったかを返す。荒れ地と最初の1回は入ったとみなさない
func (sys *AreaEntrySystem) observe(area overworld.Area) bool {
	entered := sys.observed && area.Key != sys.last && area.Key != ""
	sys.last, sys.observed = area.Key, true
	return entered
}
