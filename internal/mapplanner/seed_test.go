package mapplanner

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
)

func TestSeedReproducibility(t *testing.T) {
	t.Parallel()
	// 同じシードで複数回マップを生成して同じ結果になることを確認
	const testSeed uint64 = 12345
	width, height := gc.Tile(20), gc.Tile(20)

	// 1回目の生成
	chain1, err := NewSmallRoomPlanner(width, height, testSeed)
	require.NoError(t, err)
	chain1.PlanData.RawMaster = CreateTestRawMaster()
	err = chain1.Plan()
	require.NoError(t, err)
	tiles1 := make([]raw.TileRaw, len(chain1.PlanData.Tiles))
	copy(tiles1, chain1.PlanData.Tiles)
	rooms1 := make([]gc.Rect, len(chain1.PlanData.Rooms))
	copy(rooms1, chain1.PlanData.Rooms)

	// 2回目の生成（同じシード）
	chain2, err := NewSmallRoomPlanner(width, height, testSeed)
	require.NoError(t, err)
	chain2.PlanData.RawMaster = CreateTestRawMaster()
	err = chain2.Plan()
	require.NoError(t, err)
	tiles2 := make([]raw.TileRaw, len(chain2.PlanData.Tiles))
	copy(tiles2, chain2.PlanData.Tiles)
	rooms2 := make([]gc.Rect, len(chain2.PlanData.Rooms))
	copy(rooms2, chain2.PlanData.Rooms)

	// タイルが完全に一致することを確認
	if len(tiles1) != len(tiles2) {
		t.Errorf("タイル数が異なります。1回目: %d, 2回目: %d", len(tiles1), len(tiles2))
	}
	for i := range tiles1 {
		if tiles1[i].Name != tiles2[i].Name {
			t.Errorf("タイル[%d]が異なります。1回目: %v, 2回目: %v", i, tiles1[i], tiles2[i])
		}
	}

	// 部屋が完全に一致することを確認
	if len(rooms1) != len(rooms2) {
		t.Errorf("部屋数が異なります。1回目: %d, 2回目: %d", len(rooms1), len(rooms2))
	}
	for i := range rooms1 {
		if rooms1[i] != rooms2[i] {
			t.Errorf("部屋[%d]が異なります。1回目: %v, 2回目: %v", i, rooms1[i], rooms2[i])
		}
	}
}

func TestDifferentSeeds(t *testing.T) {
	t.Parallel()
	// 異なるシードで異なる結果になることを確認
	width, height := gc.Tile(20), gc.Tile(20)

	// シード1で生成
	chain1, err := NewSmallRoomPlanner(width, height, 11111)
	require.NoError(t, err)
	chain1.PlanData.RawMaster = CreateTestRawMaster()
	err = chain1.Plan()
	require.NoError(t, err)

	// シード2で生成
	chain2, err := NewSmallRoomPlanner(width, height, 22222)
	require.NoError(t, err)
	chain2.PlanData.RawMaster = CreateTestRawMaster()
	err = chain2.Plan()
	require.NoError(t, err)

	// 部屋数が異なる可能性が高い（必ずしも異なるとは限らないが）
	// 少なくともいくつかのタイルは異なるはず
	differentTiles := 0
	for i := range chain1.PlanData.Tiles {
		if chain1.PlanData.Tiles[i].Name != chain2.PlanData.Tiles[i].Name {
			differentTiles++
		}
	}

	if differentTiles == 0 {
		t.Error("異なるシードなのにマップが完全に一致しています")
	}
}

func TestRandomSourceDeterministic(t *testing.T) {
	t.Parallel()
	// RandomSourceが決定論的であることを確認
	const seed uint64 = 99999

	// 同じシードから作成した2つのRandomSource
	rs1 := rand.New(rand.NewPCG(seed, seed+1))
	rs2 := rand.New(rand.NewPCG(seed, seed+1))

	// 同じ順序で同じ値を生成することを確認
	for i := 0; i < 100; i++ {
		val1 := rs1.IntN(1000)
		val2 := rs2.IntN(1000)
		if val1 != val2 {
			t.Errorf("反復%dで異なる値が生成されました。val1: %d, val2: %d", i, val1, val2)
		}
	}

	// Float64も確認
	for i := 0; i < 100; i++ {
		val1 := rs1.Float64()
		val2 := rs2.Float64()
		if val1 != val2 {
			t.Errorf("反復%dで異なるFloat64値が生成されました。val1: %f, val2: %f", i, val1, val2)
		}
	}
}
