package systems

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// weatherSpellSeedSalt はスペル長の乱数を次の天候の抽選と別ストリームにするための塩。
const weatherSpellSeedSalt uint64 = 0x5715_9EA7_4E12_3D5B

// WeatherSystem は世界全体の天候スペルを進めるシステム。スペルが尽きたら次の天候を引き直す。
type WeatherSystem struct{}

// String はシステム名を返す。
func (sys *WeatherSystem) String() string {
	return "WeatherSystem"
}

// Update は現在のスペルが尽きていれば次の天候へ遷移する。まだ続くなら何もしない。
func (sys *WeatherSystem) Update(world w.World) error {
	gt := query.GetGameTime(world)
	weather := query.GetWeather(world)
	if gt.TotalTurns < weather.UntilTurn {
		return nil
	}

	// 共有 RNG を消費すると下流の敵AIや loot の乱数までずれるので、世界生成の RunSeed とターンから
	// 専用ストリームを引く。RunSeed はテストやリプレイで固定されるが Config.Seed は都度乱数になる
	band := query.GetSeamlessBand(world)
	if band == nil {
		return nil
	}
	season := gt.GetSeason()
	northDepth := playerNorthDepth(world)
	// 次の天候とスペル長で別ストリームを引く。1つの rng を順に消費すると、NextWeather の抽選が
	// 内部で何回引くかに RollSpellTurns の入力が依存して決定論が崩れるため、塩で分ける
	turn := uint64(gt.TotalTurns)
	rngNext := rand.New(rand.NewPCG(band.RunSeed, turn))
	rngSpell := rand.New(rand.NewPCG(band.RunSeed^weatherSpellSeedSalt, turn))

	prev := weather.Current
	next := query.NextWeather(prev, season, northDepth, rngNext)
	weather.Current = next
	weather.UntilTurn = gt.TotalTurns + query.RollSpellTurns(next, season, northDepth, rngSpell)

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
