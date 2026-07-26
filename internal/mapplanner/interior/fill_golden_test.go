package interior

import (
	"testing"

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

// houseContent は民家を模した Content。既存スプライト、ベッド・机・椅子・棚・ランタンでほぼ成立する。
// ベッドは奥、食卓の机と椅子は中央、棚とランタンは壁際、という住居の定石を placement で宣言する。
func houseContent() Content {
	return Content{
		ID: "house",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "bed", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "table", Placement: PlaceCenter, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "chair", Placement: PlaceCenter, Amount: Dice{Bonus: 3}},
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
