package interior

import "github.com/kijimaD/ruins/internal/consts"

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
type GroupStyle string

const (
	// PickEach は Items を全部置く。店を店たらしめる保証枠
	PickEach GroupStyle = "pick_each"
	// PickOne は Items から重みで1つ選ぶ。変種の抽選
	PickOne GroupStyle = "pick_one"
	// PickN は Items から重複なく N 個選ぶ。個数は Group.Pick が持つ
	PickN GroupStyle = "pick_n"
)

// Stuff は1つの配置指示。何を・いくつ・どこへ置くか。Satellites があれば anchor と衛星を1回で束ねて置く。
type Stuff struct {
	Kind       StuffKind
	Ref        string      // 家具型や戦利品テーブルの参照名
	Weight     int         // PickOne / PickN の抽選重み。0 は 1 とみなす
	Amount     consts.Dice // 置く個数
	Placement  Placement   // どこへ置くか。空なら PlaceFullArea 相当
	Satellites []Satellite // anchor 相対に一緒に置く衛星。机に対する椅子など
}

// Satellite は anchor 相対に一緒に置く衛星。抽選の単位を単品でなく束にすることで、机だけあって椅子が無い
// といった抽選事故を構造で防ぐ。
type Satellite struct {
	Kind    StuffKind
	Ref     string
	Offsets []Vec // anchor 相対の候補座標。前から試し、置ければ確定、尽きたら諦める
}

// Group は抽選単位の束。Style で保証セットとランダム充填を分ける。
type Group struct {
	Style GroupStyle
	Pick  int // PickN のときの選ぶ個数。PickEach / PickOne では無視する
	Items []Stuff
}

// Content は「どういう部屋に・何を置くか」の宣言。Groups を記述順に解決する。
type Content struct {
	ID     string
	Groups []Group
}

// Selection は解決済みの1配置指示。Group 解決の結果で、まだ座標を持たない。placement 段が座標を与える。
type Selection struct {
	Kind       StuffKind
	Ref        string
	Count      int
	Placement  Placement
	Satellites []Satellite // anchor ごとに一緒に置く衛星。placement 段が anchor 相対に置く
}
