package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

// putStack は player のバックパックへ id/name の同種アイテムを count 個置き、代表を返す。
// 個数は1個1エンティティで表すので count 個のエンティティを作る。name が空なら Name を付けない。
func putStack(world w.World, player ecs.Entity, id string, name string, count int) ecs.Entity {
	var rep ecs.Entity
	for range count {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		if name != "" {
			world.Components.Name.Add(e, &gc.Name{Name: name})
		}
		world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: player})
		rep = e
	}
	return rep
}

func TestFormatItemName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		itemName  string
		itemCount int
		want      string
	}{
		{
			name:      "個数が1のアイテムは名前のみ",
			itemName:  "パン",
			itemCount: 1,
			want:      "パン",
		},
		{
			name:      "個数が10のアイテムは個数付き",
			itemName:  "パン",
			itemCount: 10,
			want:      "10 パン",
		},
		{
			name:      "個数が99のアイテムは個数付き",
			itemName:  "矢",
			itemCount: 99,
			want:      "99 矢",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			world := testutil.InitTestWorld(t)
			player := world.ECS.NewEntity()
			// 同種を itemCount 個バックパックへ置く。個数は位置スタックから導出される
			rep := putStack(world, player, tt.itemName, tt.itemName, tt.itemCount)

			got := FormatItemName(world, rep)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("Nameコンポーネントがない場合はUnknown Item", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		// Name を付けずに5個置く。名前は Unknown Item にフォールバックし個数だけ付く
		rep := putStack(world, player, "unknown", "", 5)

		got := FormatItemName(world, rep)
		assert.Equal(t, "5 Unknown Item", got)
	})

	t.Run("両方のコンポーネントがない場合", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)

		// コンポーネントなしのエンティティ
		itemEntity := world.ECS.NewEntity()

		got := FormatItemName(world, itemEntity)
		assert.Equal(t, "Unknown Item", got)
	})
}
