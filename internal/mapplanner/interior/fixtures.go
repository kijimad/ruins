package interior

import "github.com/kijimaD/ruins/internal/consts"

// 束什器。anchor に衛星を相対配置した Stuff を返す。机だけあって椅子が中央に縦並びする散布事故を、束で
// 1つの scene に見せて防ぐ。レシピはこの束を1エントリとして抽選に入れる。署名什器は部屋の役割を一目で読ませ、可読性の芯になる。

// diningTable は椅子を四辺へ束ねた食卓の Stuff。机を anchor に上下左右へ椅子の衛星を配り、机だけあって
// 椅子が中央に縦並びする散布事故を anchor 相対の束で防ぐ。各椅子は正面の辺を第一候補に、埋まっていれば
// 両隣の斜めへ回り込む。
func diningTable(placement Placement) Stuff {
	chair := func(offs ...Vec) Satellite {
		return Satellite{Kind: KindFurniture, Ref: "chair", Offsets: offs}
	}
	return Stuff{
		Kind: KindFurniture, Ref: "table", Placement: placement, Amount: consts.Dice{Bonus: 1},
		Satellites: []Satellite{
			chair(Vec{X: 0, Y: -1}, Vec{X: -1, Y: -1}, Vec{X: 1, Y: -1}),
			chair(Vec{X: 0, Y: 1}, Vec{X: -1, Y: 1}, Vec{X: 1, Y: 1}),
			chair(Vec{X: -1, Y: 0}, Vec{X: -1, Y: -1}, Vec{X: -1, Y: 1}),
			chair(Vec{X: 1, Y: 0}, Vec{X: 1, Y: -1}, Vec{X: 1, Y: 1}),
		},
	}
}

// bedSet は寝床の一角を束ねる寝室の署名 fixture。ベッドを奥へ置き、脇の空いた隣へクローゼットを寄せる。
// ベッドとクローゼットを個別に散らすと部屋の別々の壁へ離れて寝室に見えないので、束で1つの寝床に見せる。
// クローゼットは4方向を順に試し、壁の向きに依らずベッドの空いた隣へ回り込む。
func bedSet() Stuff {
	return Stuff{
		Kind: KindFurniture, Ref: "bed", Placement: PlaceFarFromDoor, Amount: consts.Dice{Bonus: 1},
		Satellites: []Satellite{
			{Kind: KindFurniture, Ref: "closet", Offsets: []Vec{{X: 1}, {X: -1}, {Y: -1}, {Y: 1}}},
		},
	}
}

// loungeSet は寛ぎの一角を束ねる居間の fixture。ソファを壁際に置き、脇の空いた隣へ観葉を寄せる。食卓と
// 対になる居間の主家具で、PickOne でどちらが来るかを seed に委ねると、同じ居間が続かない。
func loungeSet() Stuff {
	return Stuff{
		Kind: KindFurniture, Ref: "sofa", Placement: PlaceWall, Amount: consts.Dice{Bonus: 1},
		Satellites: []Satellite{
			{Kind: KindDecor, Ref: "plant", Offsets: []Vec{{X: 1}, {X: -1}, {Y: -1}, {Y: 1}}},
		},
	}
}

// kitchenCounter は調理台の一列を束ねる台所の署名 fixture。流し台を壁際に置き、食器棚を横へ連ねて
// カウンターの列に見せる。流しと棚を別々に散らすと台所と分からないので、束で調理台の連なりに見せる。
// 食器棚は水平の隣を優先し、横壁沿いなら一列に、縦壁沿いなら anchor の内側へ回り込む。
func kitchenCounter() Stuff {
	return Stuff{
		Kind: KindFurniture, Ref: "sink", Placement: PlaceWall, Amount: consts.Dice{Bonus: 1},
		Satellites: []Satellite{
			{Kind: KindFurniture, Ref: "pantry", Offsets: []Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}},
			{Kind: KindFurniture, Ref: "pantry", Offsets: []Vec{{X: 2}, {X: -2}, {Y: 2}, {Y: -2}}},
		},
	}
}
