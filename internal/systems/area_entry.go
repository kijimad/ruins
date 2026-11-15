package systems

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// AreaEntrySystem は地上でプレイヤーが別の区域に入ったら、区域の名前をスプラッシュで出す。
// 前回の観測から1ターンずつ続いているときだけ比べるので、開始・ロード・ダンジョンからの帰還の直後は出さない
type AreaEntrySystem struct {
	last     overworld.AreaKey
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

// observe はターン turn のエリアを記憶し、直前の区域と別の区域に入ったかを返す。区域でない場所に
// 出ても直前の区域は忘れないので、区域の境目を行き来しても続けて出さない。system は world より長く
// 生きてロードをまたぐので、前回の観測から続いていなければ、前のプレイや別の場所の区域と比べない
func (sys *AreaEntrySystem) observe(area overworld.Area, turn consts.Turn) bool {
	continuous := sys.observed && turn == sys.lastTurn+1
	if !continuous {
		sys.last = overworld.AreaKey{}
	}
	district := area.Key != overworld.AreaKey{}
	entered := continuous && district && area.Key != sys.last
	if district {
		sys.last = area.Key
	}
	sys.lastTurn, sys.observed = turn, true
	return entered
}
