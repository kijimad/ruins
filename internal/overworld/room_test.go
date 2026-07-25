package overworld

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitRooms_全部屋がドアで連結する(t *testing.T) {
	t.Parallel()

	// 多くの seed と大きさで、分割された部屋が孤立せず必ず連結することを固定する
	for seed := uint64(1); seed < 200; seed++ {
		rng := rand.New(rand.NewPCG(seed, 0))
		r := rrect{x0: 0, y0: 0, x1: 12 + consts.Tile(rng.IntN(8)), y1: 12 + consts.Tile(rng.IntN(8))}
		rooms, walls := subdivideBuilding(rng, r)
		require.NotEmpty(t, rooms)
		assert.Truef(t, roomsConnected(rooms, walls), "seed=%d 全部屋がドアで連結する", seed)
	}
}

func TestSplitRooms_最小サイズを下回る部屋を作らない(t *testing.T) {
	t.Parallel()

	for seed := uint64(1); seed < 200; seed++ {
		rng := rand.New(rand.NewPCG(seed, 0))
		r := rrect{x0: 0, y0: 0, x1: 15, y1: 15}
		rooms, _ := subdivideBuilding(rng, r)
		for _, room := range rooms {
			assert.GreaterOrEqualf(t, room.width(), cityMinRoom, "seed=%d 部屋幅が最小以上", seed)
			assert.GreaterOrEqualf(t, room.height(), cityMinRoom, "seed=%d 部屋高が最小以上", seed)
		}
	}
}

func TestSplitRooms_大部屋は分割される(t *testing.T) {
	t.Parallel()

	// 十分大きい部屋は少なくとも2部屋に割れる。大部屋スカスカの解消を担保する
	split := 0
	for seed := uint64(1); seed < 50; seed++ {
		rng := rand.New(rand.NewPCG(seed, 0))
		rooms, _ := subdivideBuilding(rng, rrect{x0: 0, y0: 0, x1: 13, y1: 13})
		if len(rooms) >= 2 {
			split++
		}
	}
	assert.Equal(t, 49, split, "13x13 の部屋はどの seed でも複数室に割れる")
}
