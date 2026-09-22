package systems

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// WeatherSystem は世界全体の天候スペルを進めるシステム。天候はスペルすなわち「種類と続く長さ」で持ち、
// スペルが尽きた瞬間に次の天候を引いて新しい長さを設定する。1日ごとでなく可変長で移ろう。
type WeatherSystem struct{}

// String はシステム名を返す。
func (sys *WeatherSystem) String() string {
	return "WeatherSystem"
}

// Update は現在のスペルが尽きていれば次の天候へ遷移する。まだ続くなら何もしない。
// 次の天候の分布は季節と場所すなわちプレイヤーの北への奥行きで変わり、奥ほど厳しい天候へ寄る。
func (sys *WeatherSystem) Update(world w.World) error {
	gt := query.GetGameTime(world)
	weather := query.GetWeather(world)
	if gt.TotalTurns < weather.UntilTurn {
		return nil
	}

	// 共有 RNG を消費すると下流の敵AIや loot の乱数までずれる。世界生成の RunSeed とターンから専用
	// ストリームを引き、決定論的かつ下流非干渉にする。Config.Seed でなく RunSeed を使うのは、テストや
	// リプレイが RunSeed だけを固定し Config.Seed は都度乱数にするため。帯が無ければ進めない
	band := query.GetSeamlessBand(world)
	if band == nil {
		return nil
	}
	season := gt.GetSeason()
	northDepth := playerNorthDepth(world)
	rng := rand.New(rand.NewPCG(band.RunSeed, uint64(gt.TotalTurns)))

	prev := weather.Current
	next := query.NextWeather(prev, season, northDepth, rng)
	weather.Current = next
	weather.UntilTurn = gt.TotalTurns + query.RollSpellTurns(next, season, northDepth, rng)

	if next != prev {
		name := query.T(world, next.String())
		gamelog.New(query.GetGameLog(world)).
			Markup(gamelog.Tag("system", query.T(world, "The weather changed to %s.", name))).
			Log()
		// 視程が動いたら、移動を待たずその場で視界を再計算させる
		if next.Effect().VisionDelta != prev.Effect().VisionDelta {
			query.GetVisionState(world).RequestUpdate()
		}
	}
	return nil
}

// playerNorthDepth はプレイヤーの北への奥行きを返す。プレイヤーや座標が無ければ0で穏やかに倒す。
func playerNorthDepth(world w.World) int {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return 0
	}
	if !world.Components.GridElement.Has(player) {
		return 0
	}
	return query.NorthDepthChunks(world, world.Components.GridElement.Get(player).Y)
}
