package interior

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// Stuff は1つの配置指示。何を・いくつ・どこへ置くか。Satellites があれば anchor と衛星を1回で束ねて置く。
type Stuff struct {
	Kind       rawschema.StuffKind
	Ref        string              // 家具型や戦利品テーブルの参照名
	Weight     int                 // PickOne / PickN の抽選重み。0 は 1 とみなす
	Amount     consts.Dice         // 置く個数
	Placement  rawschema.Placement // どこへ置くか。PlaceDefault なら全域へ落ちる
	Satellites []Satellite         // anchor 相対に一緒に置く衛星。机に対する椅子など
	LootTable  string              // 収納家具の中身を引く itemTables の id。収納でなければ空
}

// Satellite は anchor 相対に一緒に置く衛星。抽選の単位を単品でなく束にすることで、机だけあって椅子が無い
// といった抽選事故を構造で防ぐ。
type Satellite struct {
	Kind      rawschema.StuffKind
	Ref       string
	Offsets   []Vec // anchor 相対の候補座標。前から試し、置ければ確定、尽きたら諦める
	LootTable string
}

// Group は抽選単位の束。Style で保証セットとランダム充填を分ける。
type Group struct {
	Style rawschema.GroupStyle
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
	Kind       rawschema.StuffKind
	Ref        string
	Count      int
	Placement  rawschema.Placement
	Satellites []Satellite // anchor ごとに一緒に置く衛星。placement 段が anchor 相対に置く
	LootTable  string
}
