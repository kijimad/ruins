package interior

import "github.com/kijimaD/ruins/internal/oapi"

// Placed は座標の付いた1配置。FillRoom の出力で、レンダラや ECS への spawn の入力になる。
type Placed struct {
	Kind      oapi.StuffKind
	Ref       string
	Pos       Vec
	LootTable string // 収納家具の中身を引く itemTables の id。収納でなければ空
}

// FillRoom は content を解決し、各 Selection を placement 意味論に従って部屋のタイルへ置く。
// seed 由来の決定的生成で、同じ引数なら完全に一致し再訪で一致する。占有を追跡し stuff を重ねない。
// 占有 map は抽選でなく membership 判定にのみ使うので決定性を損なわない。
func FillRoom(seed uint64, room Room, content Content) []Placed {
	occupied := make(map[Vec]bool)
	for _, w := range room.Walls {
		occupied[w] = true
	}
	placed := placeSelections(seed, room, content.Resolve(seed), occupied)
	// 塞がり防止。通路を塞ぐ家具を撤回し、戸口から全床へ到達できるようにする
	return repairReachability(room, placed)
}

// Flavor は既存の配置の隙間へ施設別の flavor decor を足す。戦利品を増やさず character を与え、空き箱
// 部屋を無くすための post-pass。flavor は装飾で通行を阻まないので到達性修復は要らない。蝋燭の輪のような
// 束も置ける。既存 placed を占有として避け、残った床へ置く。
func Flavor(seed uint64, room Room, placed []Placed, flavor Content) []Placed {
	occupied := occupiedSet(placed)
	for _, w := range room.Walls {
		occupied[w] = true
	}
	fseed := childSeed(seed, 7_000_000)
	extra := placeSelections(fseed, room, flavor.Resolve(fseed), occupied)
	return append(placed, extra...)
}

// placeSelections は解決済みの Selection 列を部屋のタイルへ置く。anchor を density 場で選び、衛星を
// anchor 相対に束ねる。occupied は呼び出し側から渡し、既存配置を避けさせる。FillRoom と Flavor が共有する。
func placeSelections(seed uint64, room Room, selections []Selection, occupied map[Vec]bool) []Placed {
	placed := make([]Placed, 0, len(selections))
	for i, sel := range selections {
		// 置き方は解決段で具体化済み。配置の seed は解決の seed と別枠にする
		s := childSeed(seed, 1_000_000+i)
		for _, t := range selectTiles(room, sel.Placement, occupied, s, sel.Count) {
			occupied[t] = true
			placed = append(placed, Placed{Kind: sel.Kind, Ref: sel.Ref, Pos: t, LootTable: sel.LootTable})
			// anchor と一緒に衛星を束で置く。机に対する椅子や蝋燭の輪など
			for _, sat := range sel.Satellites {
				if pos, ok := placeSatellite(room, occupied, t, sat); ok {
					occupied[pos] = true
					placed = append(placed, Placed{Kind: sat.Kind, Ref: sat.Ref, Pos: pos, LootTable: sat.LootTable})
				}
			}
		}
	}
	return placed
}

// placeSatellite は anchor から sat.Offsets を順に試し、部屋の内側の空きタイルに置ければその座標を返す。
// 候補が尽きたら諦める。前から試すので Offsets が優先順、机の四辺のうち空いた辺へ椅子が回り込む。戸口とその
// 直前は避ける。ここを塞ぐと部屋の唯一の入口が家具で埋まり、到達性修復は塞がれた戸口内側から flood を始められず
// 空振りして、部屋ごと入れなくなる softlock になる。anchor の selectTiles と同じ除外を衛星にも掛ける。
func placeSatellite(room Room, occupied map[Vec]bool, anchor Vec, sat Satellite) (Vec, bool) {
	for _, off := range sat.Offsets {
		p := Vec{X: anchor.X + off.X, Y: anchor.Y + off.Y}
		if room.Rect.containsInterior(p) && !occupied[p] && !isDoorwayAdjacent(room, p) {
			return p, true
		}
	}
	return Vec{}, false
}
