package interior

// Placed は座標の付いた1配置。FillRoom の出力で、レンダラや ECS への spawn の入力になる。
type Placed struct {
	Kind StuffKind
	Ref  string
	Pos  Vec
}

// FillRoom は content を解決し、各 Selection を placement 意味論に従って部屋のタイルへ置く。
// seed 由来の決定的生成で、同じ引数なら完全に一致し再訪で一致する。占有を追跡し stuff を重ねない。
// 占有 map は抽選でなく membership 判定にのみ使うので決定性を損なわない。
func FillRoom(seed uint64, room Room, content Content) []Placed {
	selections := content.Resolve(seed)
	occupied := make(map[Vec]bool)
	placed := make([]Placed, 0, len(selections))
	for i, sel := range selections {
		p := sel.Placement
		if p == "" {
			p = PlaceFullArea
		}
		// 配置の seed は解決の seed と別枠にし、片方を変えても他方が動かないようにする
		s := childSeed(seed, 1_000_000+i)
		for _, t := range selectTiles(room, p, occupied, s, sel.Count) {
			occupied[t] = true
			placed = append(placed, Placed{Kind: sel.Kind, Ref: sel.Ref, Pos: t})
			// anchor と一緒に衛星を束で置く。机に対する椅子など
			for _, sat := range sel.Satellites {
				if pos, ok := placeSatellite(room, occupied, t, sat); ok {
					occupied[pos] = true
					placed = append(placed, Placed{Kind: sat.Kind, Ref: sat.Ref, Pos: pos})
				}
			}
		}
	}
	// 塞がり防止。通路を塞ぐ家具を撤回し、戸口から全床へ到達できるようにする
	return repairReachability(room, placed)
}

// placeSatellite は anchor から sat.Offsets を順に試し、部屋の内側の空きタイルに置ければその座標を返す。
// 候補が尽きたら諦める。前から試すので Offsets が優先順、机の四辺のうち空いた辺へ椅子が回り込む。
func placeSatellite(room Room, occupied map[Vec]bool, anchor Vec, sat Satellite) (Vec, bool) {
	for _, off := range sat.Offsets {
		p := Vec{X: anchor.X + off.X, Y: anchor.Y + off.Y}
		if room.Rect.containsInterior(p) && !occupied[p] {
			return p, true
		}
	}
	return Vec{}, false
}
