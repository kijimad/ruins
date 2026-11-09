package interior

import "github.com/kijimaD/ruins/internal/consts"

// hero 部屋。大多数の背景の建物は密度と散らかりの揺らぎで足りるが、稀に1棟だけ記憶に残る見せ場を置く。
// 異質なものは孤立しているほど記憶に残るので、landmark 級は狭い確率で1棟に1つに絞る。
// 中身は 報酬46/雰囲気25/リスクリワード15/危険14 の予算配分で引き、主室の中央へ1つだけ据える。

// heroKind は hero 部屋の目玉の種類。
type heroKind string

const (
	heroReward     heroKind = "reward"
	heroAtmosphere heroKind = "atmosphere"
	heroRiskReward heroKind = "risk_reward"
	heroDanger     heroKind = "danger"
)

// heroCenterpiece は建物が hero なら主室に据える目玉 prop の Ref を返す。大半の建物は hero でない。予算配分で
// 4種を引く。報酬は宝箱、雰囲気は鳥居や蓄音機の landmark、リスクリワードは檻の中の獲物、危険は毒樽。
func heroCenterpiece(seed uint64) (string, bool) {
	if childSeed(seed, 13_000_000)%12 != 0 {
		return "", false // 12棟に1棟だけ見せ場を持つ。rare を孤立させる
	}
	switch pickChoice(childSeed(seed, 13_100_000), []choice[heroKind]{
		{heroReward, 46}, {heroAtmosphere, 25}, {heroRiskReward, 15}, {heroDanger, 14},
	}) {
	case heroReward:
		return "chest", true
	case heroAtmosphere:
		return pickOne(childSeed(seed, 13_200_000), []string{"torii", "phonograph", "komainu", "pillar"}), true
	case heroRiskReward:
		return "cage", true
	case heroDanger:
		return "poison", true
	}
	panic("unknown heroKind")
}

// heroSpot は目玉を据えるタイルを返す。廊下を除く最大の部屋の中央。中央が内側床でなければ置かない。
func heroSpot(s Site) (Vec, bool) {
	best, bestArea := -1, consts.Tile(0)
	for i, hr := range s.Rooms {
		if hr.Role == roleCorridor {
			continue
		}
		if a := hr.Room.Rect.W * hr.Room.Rect.H; a > bestArea {
			best, bestArea = i, a
		}
	}
	if best < 0 {
		return Vec{}, false
	}
	c := s.Rooms[best].Room.Rect.center()
	if s.Rooms[best].Room.Rect.containsInterior(c) {
		return c, true
	}
	return Vec{}, false
}
