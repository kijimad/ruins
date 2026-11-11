package interior

import "github.com/kijimaD/ruins/internal/consts"

// hero 部屋。大多数の背景の建物は密度と散らかりの揺らぎで足りるが、稀に1棟だけ記憶に残る見せ場を置く。
// 異質なものは孤立しているほど記憶に残るので、landmark 級は狭い確率で1棟に1つに絞る。
// 中身は raw の interiorProfile.heroes の予算配分で引き、主室の中央へ1つだけ据える。

// heroCenterpiece は建物が hero なら主室に据える目玉 prop の Ref を返す。大半の建物は hero でない。予算配分で
// 種類を引き、その種類の Ref から1つを一様に引く。報酬は宝箱、雰囲気は鳥居や蓄音機の landmark、
// リスクリワードは檻の中の獲物、危険は毒樽。
func heroCenterpiece(t profileTable, seed uint64) (string, bool) {
	if childSeed(seed, 13_000_000)%t.heroOneIn != 0 {
		return "", false // heroOneIn 棟に1棟だけ見せ場を持つ。rare を孤立させる
	}
	refs := pickChoice(childSeed(seed, 13_100_000), t.heroes)
	return pickOne(childSeed(seed, 13_200_000), refs), true
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
