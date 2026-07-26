package interior

// Age は生成直後の新品の内装に時間の層を刻む。略奪 → 生活痕 → 廃墟化の3 post-pass を順に適用する。
// 各 pass は seed 由来で決定的。ruins は廃墟が主題なので、この層が「新品の店」を「打ち捨てられた店」に
// 変える。配置(FillRoom)と時間(Age)を分けるので、新品と経年の両方を同じ器で VRT にできる。
//
// 略奪は anchor 温存・衛星欠損で表す。構造(家具)を残し中身(戦利品)を欠くと、部屋の可読性を保ったまま
// 「荒らされた」を出せる。廃墟化は瓦礫を撒くが、瓦礫は装飾で通行は阻まないので到達性は保たれる。
func Age(seed uint64, room Room, placed []Placed) []Placed {
	placed = applyLooting(childSeed(seed, 1), placed)
	placed = applyWear(childSeed(seed, 2), room, placed)
	placed = applyDecay(childSeed(seed, 3), room, placed)
	return placed
}

// applyLooting は戦利品の一部を撤去する。家具(anchor)は残し戦利品(衛星)を欠かせる。
func applyLooting(seed uint64, placed []Placed) []Placed {
	out := make([]Placed, 0, len(placed))
	for i, p := range placed {
		if p.Kind == KindLoot && dropChance(seed, i, 55) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// applyWear は生活痕を足す。家具の一部が隣接する空きタイルへ小物を落とす。散乱が乱数でなく
// 「そこに家具があった」因果になる。
func applyWear(seed uint64, room Room, placed []Placed) []Placed {
	occupied := occupiedSet(placed)
	added := make([]Placed, 0)
	for i, p := range placed {
		if p.Kind != KindFurniture || !dropChance(childSeed(seed, i), 0, 35) {
			continue
		}
		for _, n := range neighbors4(p.Pos) {
			if room.Rect.containsInterior(n) && !occupied[n] {
				occupied[n] = true
				added = append(added, Placed{Kind: KindDecor, Ref: "debris", Pos: n})
				break
			}
		}
	}
	return append(placed, added...)
}

// applyDecay は廃墟化。壁際の露出が高いタイルに瓦礫の種を撒き、固定K世代の CA で近傍へ伝播させ、
// しきい値で瓦礫を置く。反復手法を固定手数へ還元した決定的な崩壊伝播。瓦礫は装飾で通行を阻まない。
func applyDecay(seed uint64, room Room, placed []Placed) []Placed {
	rubble := make(map[Vec]bool)
	// 種。壁際 かつ hash が低いタイル。露出の高い縁から朽ちる
	for _, t := range room.Rect.interiorTiles() {
		if nextToPerimeter(room.Rect, t) && norm01(hashTile(seed, t)) < 0.28 {
			rubble[t] = true
		}
	}
	// 固定2世代の CA 伝播。瓦礫近傍が2以上のタイルは hash で瓦礫化する
	for gen := range 2 {
		next := make(map[Vec]bool, len(rubble))
		for _, t := range room.Rect.interiorTiles() {
			if rubble[t] {
				next[t] = true
				continue
			}
			c := 0
			for _, n := range neighbors4(t) {
				if rubble[n] {
					c++
				}
			}
			if c >= 2 && norm01(hashTile(childSeed(seed, gen+10), t)) < 0.5 {
				next[t] = true
			}
		}
		rubble = next
	}
	// 空きかつ戸口前でないタイルへ瓦礫を置く。戸口は塞がない
	occupied := occupiedSet(placed)
	added := make([]Placed, 0)
	for _, t := range room.Rect.interiorTiles() {
		if rubble[t] && !occupied[t] && !isDoorwayAdjacent(room, t) {
			added = append(added, Placed{Kind: KindDecor, Ref: "rubble", Pos: t})
		}
	}
	return append(placed, added...)
}

// occupiedSet は配置済みタイルの集合。membership 判定にのみ使う。
func occupiedSet(placed []Placed) map[Vec]bool {
	m := make(map[Vec]bool, len(placed))
	for _, p := range placed {
		m[p.Pos] = true
	}
	return m
}

// dropChance は index i について pct% で真を返す決定的判定。childSeed で index ごとに独立させる。
func dropChance(seed uint64, i, pct int) bool {
	return int(childSeed(seed, i)%100) < pct
}
