package interior

// StuffKind は配置指示の種別。家具・戦利品・敵・装飾・罠を同じ器で扱う。
// 実体は文字列。%v やログで数値でなく種別名が出て、デバッグで読みやすい。
type StuffKind string

const (
	// KindFurniture は家具。占有し移動を阻む
	KindFurniture StuffKind = "furniture"
	// KindLoot は戦利品。コンテナや床置き
	KindLoot StuffKind = "loot"
	// KindBeing は敵・住人
	KindBeing StuffKind = "being"
	// KindDecor は装飾。通行を阻まない小物
	KindDecor StuffKind = "decor"
	// KindTrap は罠
	KindTrap StuffKind = "trap"
)

// GroupStyle は Group の抽選方式。保証セットとランダム充填を分ける散布殺しの芯。
// 実体は文字列。
type GroupStyle string

const (
	// PickEach は Items を全部置く。店を店たらしめる保証枠。各 Stuff は Chance で個別に gate する
	PickEach GroupStyle = "pick_each"
	// PickOne は Items から重みで1つ選ぶ。変種の抽選
	PickOne GroupStyle = "pick_one"
	// PickN は Items から重複なく N 個選ぶ。個数は Group.Pick が持つ
	PickN GroupStyle = "pick_n"
)

// Dice は個数抽選。Base 個の Sides 面ダイスの和に Bonus を足す。1d3+1 は {Base:1, Sides:3, Bonus:1}。
// 定数個数は Sides<=0 とし、そのとき値は Bonus になる。5固定は {Sides:0, Bonus:5}。
type Dice struct {
	Base  int
	Sides int
	Bonus int
}

// Stuff は1つの配置指示。Stage 1 では「何を・いくつ」だけを持つ。どこへ置くか(placement)は後続 Stage。
type Stuff struct {
	Kind   StuffKind
	Ref    string // 家具型や戦利品テーブルの参照名
	Weight int    // PickOne / PickN の抽選重み。0 は 1 とみなす
	Chance int    // 0..100。PickEach でこの Stuff を置く確率。0 以下は常置
	Amount Dice   // 置く個数
}

// Group は抽選単位の束。Style で保証セットとランダム充填を分ける。
type Group struct {
	Style GroupStyle
	Pick  int // PickN のときの選ぶ個数。PickEach / PickOne では無視する
	Items []Stuff
}

// Content は「どういう部屋に・何を置くか」の宣言。Stage 1 では Groups の解決だけを担う。
// ThemeTags による施設種の抽選や RoomReq 照合は後続 Stage で足す。
type Content struct {
	ID     string
	Groups []Group
}

// Selection は解決済みの1配置。Group 解決の結果で、まだ座標を持たない。placement 段が座標を与える。
type Selection struct {
	Kind  StuffKind
	Ref   string
	Count int
}
