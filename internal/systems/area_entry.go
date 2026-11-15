package systems

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// AreaEntrySystem は地上でプレイヤーが別の区域に入ったら、区域の名前をスプラッシュで出す。
// 開始・ロード・ダンジョンからの帰還の直後に区域にいるときも出す
type AreaEntrySystem struct {
	last     string
	lastTurn consts.Turn
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
	if sys.observe(area, query.GetGameTime(world).TotalTurns) {
		lifecycle.SpawnSplashText(world, query.T(world, area.Name))
	}
	return nil
}

// Restart は直前のエリアを忘れて観測を始め直し、区域にいれば名前を出す。地上の開始とロードで呼ぶ。
// ロードした世界のターンが直前の観測と続いて見えても、前のプレイのエリアと比べないようにする
func (sys *AreaEntrySystem) Restart(world w.World) error {
	sys.observed = false
	return sys.Update(world)
}

// observe はターン turn のエリアを記憶し、区域に入ったかを返す。前回の観測から続いていなければ、
// 直前のエリアを忘れて今いる区域に入ったとみなす。system は world より長く生きてロードをまたぐので、
// 前のプレイや別の場所で記憶したエリアと比べないよう、ターンの連続を見る
func (sys *AreaEntrySystem) observe(area overworld.Area, turn consts.Turn) bool {
	// 開始時にも1回観測するので、同じターンの観測も続いているとみなす
	continuous := sys.observed && (turn == sys.lastTurn || turn == sys.lastTurn+1)
	entered := area.Key != "" && (!continuous || area.Key != sys.last)
	sys.last, sys.lastTurn, sys.observed = area.Key, turn, true
	return entered
}
