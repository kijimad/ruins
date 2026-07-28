package interior

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFlavor_同じseedで完全一致する は flavor パスの決定性を固定する。再訪一致と serde の前提。
func TestFlavor_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	room := houseSmallRoom()
	base := FillRoom(3, room, houseContent())
	first := Flavor(3, room, base, abandonedFlavor())
	for range 5 {
		require.Equal(t, first, Flavor(3, room, base, abandonedFlavor()), "同じ引数なら flavor も完全一致する")
	}
}

// TestFlavor_到達性を壊さない は flavor が装飾で通行を阻まないことを固定する。蝋燭の輪を家具へ格上げ
// すると塞ぎうるので、その退行を検知する。flavor 追加後も戸口から全床へ到達できること。
func TestFlavor_到達性を壊さない(t *testing.T) {
	t.Parallel()

	room := storeRoom()
	for seed := range uint64(30) {
		base := FillRoom(seed, room, storeContent())
		flavored := Flavor(seed, room, base, abandonedFlavor())

		blocked := blockingTiles(flavored)
		reached := reachableFloor(room, blocked)
		for _, tile := range room.Rect.interiorTiles() {
			if blocked[tile] {
				continue
			}
			require.Truef(t, reached[tile], "flavor 後も床 %v が戸口から到達できる (seed=%d)", tile, seed)
		}
	}
}
