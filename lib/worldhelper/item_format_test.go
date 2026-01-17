package worldhelper_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/testutil"
	"github.com/kijimaD/ruins/lib/worldhelper"
	"github.com/stretchr/testify/assert"
)

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
			want:      "パン(10個)",
		},
		{
			name:      "個数が99のアイテムは個数付き",
			itemName:  "矢",
			itemCount: 99,
			want:      "矢(99個)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			world := testutil.InitTestWorld(t)

			// アイテムエンティティを作成
			itemEntity := world.Manager.NewEntity()
			itemEntity.AddComponent(world.Components.Name, &gc.Name{
				Name: tt.itemName,
			})
			itemEntity.AddComponent(world.Components.Item, &gc.Item{
				Count: tt.itemCount,
			})

			got := worldhelper.FormatItemName(world, itemEntity)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("Nameコンポーネントがない場合はUnknown Item", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)

		// Nameコンポーネントなしのエンティティ
		itemEntity := world.Manager.NewEntity()
		itemEntity.AddComponent(world.Components.Item, &gc.Item{
			Count: 5,
		})

		got := worldhelper.FormatItemName(world, itemEntity)
		assert.Equal(t, "Unknown Item(5個)", got)
	})

	t.Run("両方のコンポーネントがない場合", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)

		// コンポーネントなしのエンティティ
		itemEntity := world.Manager.NewEntity()

		got := worldhelper.FormatItemName(world, itemEntity)
		assert.Equal(t, "Unknown Item", got)
	})
}
