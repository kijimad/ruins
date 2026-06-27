package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
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
			if tt.itemCount > 1 {
				itemEntity.AddComponent(world.Components.Stackable, &gc.Stackable{Count: tt.itemCount})
			}

			got := FormatItemName(world, itemEntity)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("Nameコンポーネントがない場合はUnknown Item", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)

		// Nameコンポーネントなしのエンティティ
		itemEntity := world.Manager.NewEntity()
		itemEntity.AddComponent(world.Components.Stackable, &gc.Stackable{Count: 5})

		got := FormatItemName(world, itemEntity)
		assert.Equal(t, "Unknown Item(5個)", got)
	})

	t.Run("両方のコンポーネントがない場合", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)

		// コンポーネントなしのエンティティ
		itemEntity := world.Manager.NewEntity()

		got := FormatItemName(world, itemEntity)
		assert.Equal(t, "Unknown Item", got)
	})
}
