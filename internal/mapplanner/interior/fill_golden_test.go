package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// このファイルは VRT の fixture、すなわち施設ごとの Content と Room の定義を持つ。実際の描画は
// sprite_golden_test.go の実スプライトレンダラが担う。模式図(文字+色)では「自然にその施設に見えるか」を
// 判断できないため、VRT はゲームと同じ 32px スプライト版に一本化した。

// storeContent はコンビニを模した Content。placement 意味論で「店に見える」配置を宣言する。
// 冷蔵ケースは奥、レジは入口近く、ゴンドラは中央、雑貨は壁際。
func storeContent() Content {
	return Content{
		ID: "conv_store",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "walkin_cooler", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 7}},
				{Kind: KindFurniture, Ref: "register", Placement: PlaceNearDoor, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "gondola", Placement: PlaceRow, Amount: Dice{Bonus: 10}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "snacks", Placement: PlaceCenter, Weight: 3, Amount: Dice{Base: 2, Sides: 4}},
				{Kind: KindLoot, Ref: "drinks", Placement: PlaceFarFromDoor, Weight: 2, Amount: Dice{Base: 1, Sides: 4}},
				{Kind: KindLoot, Ref: "bento", Placement: PlaceWall, Weight: 1, Amount: Dice{Base: 1, Sides: 3}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "litter", Placement: PlaceFullArea, Amount: Dice{Base: 1, Sides: 3, Bonus: 1}},
			}},
		},
	}
}

// storeRoom は入口が下辺中央の 15x10 の部屋。
func storeRoom() Room {
	return Room{
		Rect:     Rect{X: 0, Y: 0, W: 15, H: 10},
		Doorways: []Doorway{{X: 7, Y: 9}},
	}
}

// clinicContent は診療所を模した Content。同じ器に別の content を流すだけで別の施設になることを示す。
// 受付と待合椅子は入口近く、診察ベッドは奥、薬棚は壁際、という診療所の定石を placement で宣言する。
func clinicContent() Content {
	return Content{
		ID: "clinic",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "reception", Placement: PlaceNearDoor, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "waitchair", Placement: PlaceNearDoor, Amount: Dice{Bonus: 5}},
				{Kind: KindFurniture, Ref: "exam_bed", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 3}},
				{Kind: KindFurniture, Ref: "medcabinet", Placement: PlaceWall, Amount: Dice{Bonus: 3}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "meds", Placement: PlaceFarFromDoor, Weight: 2, Amount: Dice{Base: 1, Sides: 3}},
				{Kind: KindLoot, Ref: "bandage", Placement: PlaceWall, Weight: 1, Amount: Dice{Base: 1, Sides: 2}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "plant", Placement: PlaceFullArea, Amount: Dice{Bonus: 2}},
			}},
		},
	}
}

// clinicRoom は入口が下辺中央の 16x11 の部屋。
func clinicRoom() Room {
	return Room{
		Rect:     Rect{X: 0, Y: 0, W: 16, H: 11},
		Doorways: []Doorway{{X: 8, Y: 10}},
	}
}

// diningTable は椅子を四辺へ束ねた食卓の Stuff。机を anchor に、上下左右へ椅子の衛星を配る。机だけ
// 置いて椅子が中央に縦並びする事故を、anchor 相対の束で防ぐ。各椅子は正面の辺を第一候補に、埋まって
// いれば両隣の斜めへ回り込む。
func diningTable(placement Placement) Stuff {
	chair := func(offs ...Vec) Satellite {
		return Satellite{Kind: KindFurniture, Ref: "chair", Offsets: offs}
	}
	return Stuff{
		Kind: KindFurniture, Ref: "table", Placement: placement, Amount: Dice{Bonus: 1},
		Satellites: []Satellite{
			chair(Vec{X: 0, Y: -1}, Vec{X: -1, Y: -1}, Vec{X: 1, Y: -1}),
			chair(Vec{X: 0, Y: 1}, Vec{X: -1, Y: 1}, Vec{X: 1, Y: 1}),
			chair(Vec{X: -1, Y: 0}, Vec{X: -1, Y: -1}, Vec{X: -1, Y: 1}),
			chair(Vec{X: 1, Y: 0}, Vec{X: 1, Y: -1}, Vec{X: 1, Y: 1}),
		},
	}
}

// houseContent は民家を模した Content。既存スプライト、ベッド・机・椅子・棚・ランタンでほぼ成立する。
// ベッドは奥、食卓の机と椅子は中央、棚とランタンは壁際、という住居の定石を placement で宣言する。
func houseContent() Content {
	return Content{
		ID: "house",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "bed", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 1}},
				diningTable(PlaceCenter),
				{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 2}},
				{Kind: KindFurniture, Ref: "lantern", Placement: PlaceWall, Amount: Dice{Bonus: 2}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "plant", Placement: PlaceFullArea, Amount: Dice{Bonus: 2}},
			}},
		},
	}
}

// houseRoom は入口が下辺中央の 13x9 の部屋。
func houseRoom() Room {
	return Room{
		Rect:     Rect{X: 0, Y: 0, W: 13, H: 9},
		Doorways: []Doorway{{X: 6, Y: 8}},
	}
}

// TestFillRoom_同じseedで完全一致する は配置まで含めた決定性を固定する。再訪一致と serde の前提。
func TestFillRoom_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	room, content := storeRoom(), storeContent()
	first := FillRoom(42, room, content)
	for range 5 {
		require.Equal(t, first, FillRoom(42, room, content), "同じ seed なら配置も完全一致する")
	}
}

// TestFillRoom_衛星の椅子は机の隣に置かれる は anchor 付き束の不変条件を固定する。机を anchor に束ねた
// 椅子は必ず机の隣接8マスに来る。中央にバラ置きして椅子が縦並びする散布事故を、束が構造で防ぐことを守る。
func TestFillRoom_衛星の椅子は机の隣に置かれる(t *testing.T) {
	t.Parallel()

	room := Room{Rect: Rect{X: 0, Y: 0, W: 9, H: 9}, Doorways: []Doorway{{X: 4, Y: 8}}}
	content := Content{ID: "dining", Groups: []Group{{Style: PickEach, Items: []Stuff{diningTable(PlaceCenter)}}}}
	placed := FillRoom(1, room, content)

	var table Vec
	var chairs []Vec
	for _, p := range placed {
		switch p.Ref {
		case "table":
			table = p.Pos
		case "chair":
			chairs = append(chairs, p.Pos)
		}
	}
	require.NotEmpty(t, chairs, "椅子が置かれる")
	for _, c := range chairs {
		assert.LessOrEqualf(t, max(abs(c.X-table.X), abs(c.Y-table.Y)), 1, "椅子 %v は机 %v の隣接8マスに来る", c, table)
	}
}
