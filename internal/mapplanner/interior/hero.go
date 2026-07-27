package interior

// hero 部屋。大多数の背景の建物は密度と散らかりの揺らぎで足りるが、稀に1棟だけ記憶に残る見せ場を置く。
// 異質なものは孤立しているほど記憶に残るので、landmark 級は狭い確率で1棟に1つに絞る(doc L695-697)。
// 中身は 報酬46/雰囲気25/リスクリワード15/危険14 の予算配分で引き、主室の中央へ1つだけ据える。
// docs/design/20260725_70.md 追記その5 収穫5・追記その16。

// heroCenterpiece は建物が hero なら主室に据える目玉 prop の Ref を返す。大半の建物は hero でない。予算配分で
// 報酬(宝箱)・雰囲気(鳥居や蓄音機の landmark)・リスクリワード(檻の中の獲物)・危険(毒樽)を引く。
func heroCenterpiece(seed uint64) (string, bool) {
	if childSeed(seed, 13_000_000)%12 != 0 {
		return "", false // 12棟に1棟だけ見せ場を持つ。rare を孤立させる
	}
	switch r := childSeed(seed, 13_100_000) % 100; {
	case r < 46: // 報酬
		return "chest", true
	case r < 71: // 雰囲気。landmark を seed で1つ
		atmos := []string{"torii", "phonograph", "komainu", "pillar"}
		return atmos[int(childSeed(seed, 13_200_000)%uint64(len(atmos)))], true
	case r < 86: // リスクリワード。檻に入った獲物
		return "cage", true
	default: // 危険
		return "poison", true
	}
}

// heroSpot は目玉を据えるタイルを返す。廊下を除く最大の部屋の中央。中央が内側床でなければ置かない。
func heroSpot(s Site) (Vec, bool) {
	best, bestArea := -1, 0
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
