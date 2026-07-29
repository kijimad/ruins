package interior

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAge_同じseedで完全一致する は時間の層まで含めた決定性を固定する。再訪一致と serde の前提。
func TestAge_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	room := storeRoom()
	base := FillRoom(9, room, storeContent())
	first := Age(9, room, base, dmgMajor)
	for range 5 {
		require.Equal(t, first, Age(9, room, base, dmgMajor), "Age は同じ引数で完全一致する")
	}
}

// TestAge_到達性を壊さない は、廃墟化の瓦礫が装飾で通行を阻まず、経年後も戸口から全床へ到達できることを
// 固定する。瓦礫を家具に格上げすると塞ぎうるので、その退行を検知する。
func TestAge_到達性を壊さない(t *testing.T) {
	t.Parallel()

	room := storeRoom()
	for s := range uint64(30) {
		aged := Age(s, room, FillRoom(s, room, storeContent()), dmgMajor)
		blocked := blockingTiles(aged)
		reached := reachableFloor(room, blocked)
		for _, tile := range room.Rect.interiorTiles() {
			if blocked[tile] {
				continue
			}
			require.Truef(t, reached[tile], "経年後も床 %v が戸口から到達できる (seed=%d)", tile, s)
		}
	}
}
