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
		}
	}
	// 塞がり防止。通路を塞ぐ家具を撤回し、戸口から全床へ到達できるようにする
	return repairReachability(room, placed)
}
