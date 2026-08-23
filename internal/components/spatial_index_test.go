package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestNewSpatialIndex_未構築の空インデックスを返す(t *testing.T) {
	t.Parallel()

	si := NewSpatialIndex()

	assert.False(t, si.Built)
	assert.Nil(t, si.BlockPass)
	assert.Nil(t, si.Characters)
	assert.Nil(t, si.PlayerEntity)
}

func TestSpatialIndex_IsBlockPass(t *testing.T) {
	t.Parallel()

	pos := consts.Coord[consts.Tile]{X: 1, Y: 2}

	t.Run("未構築ならfalseを返す", func(t *testing.T) {
		t.Parallel()
		si := NewSpatialIndex()
		assert.False(t, si.IsBlockPass(pos))
	})

	t.Run("構築済みで登録されたタイルはtrueを返す", func(t *testing.T) {
		t.Parallel()
		si := &SpatialIndex{
			Built:     true,
			BlockPass: map[GridElement]bool{{Coord: pos}: true},
		}
		assert.True(t, si.IsBlockPass(pos))
	})

	t.Run("構築済みでも未登録のタイルはfalseを返す", func(t *testing.T) {
		t.Parallel()
		si := &SpatialIndex{
			Built:     true,
			BlockPass: map[GridElement]bool{},
		}
		assert.False(t, si.IsBlockPass(pos))
	})
}

func TestSpatialIndex_CharacterAt(t *testing.T) {
	t.Parallel()

	pos := consts.Coord[consts.Tile]{X: 3, Y: 4}

	t.Run("登録されたタイルはエンティティとtrueを返す", func(t *testing.T) {
		t.Parallel()
		world := ecs.NewWorld()
		entity := world.NewEntity()
		si := &SpatialIndex{Characters: map[GridElement]ecs.Entity{{Coord: pos}: entity}}
		got, ok := si.CharacterAt(pos)
		assert.True(t, ok)
		assert.Equal(t, entity, got)
	})

	t.Run("未登録のタイルはfalseを返す", func(t *testing.T) {
		t.Parallel()
		si := &SpatialIndex{Characters: map[GridElement]ecs.Entity{}}
		_, ok := si.CharacterAt(pos)
		assert.False(t, ok)
	})
}

func TestSpatialIndex_MoveCharacter(t *testing.T) {
	t.Parallel()

	from := consts.Coord[consts.Tile]{X: 0, Y: 0}
	to := consts.Coord[consts.Tile]{X: 1, Y: 0}

	// ecs.World の NewEntity は構造変更で並列安全でないため、ケースごとに個別の world を作る。
	t.Run("未構築なら何もしない", func(t *testing.T) {
		t.Parallel()
		world := ecs.NewWorld()
		e := world.NewEntity()
		si := NewSpatialIndex()
		si.MoveCharacter(from, to, e)
		assert.Nil(t, si.Characters)
	})

	t.Run("移動元の登録が自分自身なら削除して移動先へ登録する", func(t *testing.T) {
		t.Parallel()
		world := ecs.NewWorld()
		e := world.NewEntity()
		si := &SpatialIndex{
			Built:      true,
			Characters: map[GridElement]ecs.Entity{{Coord: from}: e},
		}
		si.MoveCharacter(from, to, e)

		_, fromOK := si.Characters[GridElement{Coord: from}]
		assert.False(t, fromOK, "移動元の登録が消える")
		gotTo, toOK := si.Characters[GridElement{Coord: to}]
		assert.True(t, toOK)
		assert.Equal(t, e, gotTo)
	})

	t.Run("移動元の登録が別キャラなら削除せず移動先だけ登録する", func(t *testing.T) {
		t.Parallel()
		world := ecs.NewWorld()
		mover := world.NewEntity()
		other := world.NewEntity()
		si := &SpatialIndex{
			Built:      true,
			Characters: map[GridElement]ecs.Entity{{Coord: from}: other},
		}
		si.MoveCharacter(from, to, mover)

		gotFrom, fromOK := si.Characters[GridElement{Coord: from}]
		assert.True(t, fromOK, "別キャラの登録は残る")
		assert.Equal(t, other, gotFrom)
		gotTo, toOK := si.Characters[GridElement{Coord: to}]
		assert.True(t, toOK)
		assert.Equal(t, mover, gotTo)
	})
}

func TestSpatialIndex_Invalidate(t *testing.T) {
	t.Parallel()

	world := ecs.NewWorld()
	player := world.NewEntity()
	si := &SpatialIndex{
		Built:        true,
		BlockPass:    map[GridElement]bool{{}: true},
		Characters:   map[GridElement]ecs.Entity{{}: player},
		PlayerEntity: &player,
		BuildCount:   3,
	}

	si.Invalidate()

	assert.False(t, si.Built)
	assert.Nil(t, si.BlockPass)
	assert.Nil(t, si.Characters)
	assert.Nil(t, si.PlayerEntity)
	assert.Equal(t, 3, si.BuildCount, "BuildCountはリセットしない")
}
