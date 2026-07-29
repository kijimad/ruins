package interior

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// このファイルは VRT の fixture、すなわち施設ごとの Content と Room の定義を持つ。実際の描画は
// sprite_golden_test.go の実スプライトレンダラが担う。模式図(文字+色)では「自然にその施設に見えるか」を
// 判断できないため、VRT はゲームと同じ 32px スプライト版に一本化した。

// storeRoom は入口が下辺中央の 15x10 の部屋。storeContent など施設 content は facility.go にある。
func storeRoom() Room {
	return Room{
		Rect:     Rect{X: 0, Y: 0, W: 15, H: 10},
		Doorways: []Doorway{{X: 7, Y: 9}},
	}
}

// clinicRoom は入口が下辺中央の 16x11 の部屋。
func clinicRoom() Room {
	return Room{
		Rect:     Rect{X: 0, Y: 0, W: 16, H: 11},
		Doorways: []Doorway{{X: 8, Y: 10}},
	}
}

// abandonedFlavor は廃墟に生活の痕を足す flavor machine の Content。絨毯・箒・散らばった蝋燭のうち2つを
// 隅や壁際へ置く。戦利品を増やさず character を与え、空き箱部屋を無くす。Flavor パスで既存配置の隙間へ
// 流し込む。production の facilityFlavor と同じく儀式の輪は置かない。
func abandonedFlavor() Content {
	return Content{
		ID: "abandoned",
		Groups: []Group{
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindDecor, Ref: "carpet", Placement: PlaceFarFromDoor, Amount: consts.Dice{Bonus: 1}},
				{Kind: KindDecor, Ref: "broom", Placement: PlaceWall, Amount: consts.Dice{Bonus: 1}},
				{Kind: KindDecor, Ref: "candle", Placement: PlaceFullArea, Amount: consts.Dice{Base: 1, Sides: 2}},
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
