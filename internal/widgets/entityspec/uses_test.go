package entityspec_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestUseHints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(world w.World, e ecs.Entity)
		// want は「用途」見出しに続けて並ぶ性質の msgid。見出しとインデントはテスト側で組む
		want []string
	}{
		{
			name: "栄養を持つ消費物は食べられる",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Consumable.Add(e, &gc.Consumable{})
				world.Components.ProvidesNutrition.Add(e, &gc.ProvidesNutrition{Amount: 10})
			},
			want: []string{"Edible"},
		},
		{
			name: "回復を持つ消費物は食べられる",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Consumable.Add(e, &gc.Consumable{})
				world.Components.ProvidesHealing.Add(e, &gc.ProvidesHealing{Amount: 10})
			},
			want: []string{"Edible"},
		},
		{
			name: "栄養も回復も持たない消費物は道具として使える",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Consumable.Add(e, &gc.Consumable{})
			},
			want: []string{"Usable"},
		},
		{
			name: "本は読める",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Book.Add(e, &gc.Book{})
			},
			want: []string{"Readable"},
		},
		{
			name: "防具は装備できる",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Wearable.Add(e, &gc.Wearable{})
			},
			want: []string{"Wearable"},
		},
		{
			name: "近接武器は装備できる",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Melee.Add(e, &gc.Melee{})
			},
			want: []string{"Wearable"},
		},
		{
			name: "遠距離武器は装備できる",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Fire.Add(e, &gc.Fire{})
			},
			want: []string{"Wearable"},
		},
		{
			name: "分解工具は物を分解できる",
			setup: func(world w.World, e ecs.Entity) {
				// monkey_wrench は raw に分解工具定義を持つ。判定は RawID から raw を引く
				world.Components.RawID.Add(e, &gc.RawID{ID: "monkey_wrench"})
			},
			want: []string{"Can disassemble items"},
		},
		{
			name: "複数の性質は表示順に並ぶ",
			setup: func(world w.World, e ecs.Entity) {
				world.Components.Consumable.Add(e, &gc.Consumable{})
				world.Components.ProvidesNutrition.Add(e, &gc.ProvidesNutrition{Amount: 10})
				world.Components.Wearable.Add(e, &gc.Wearable{})
			},
			want: []string{"Edible", "Wearable"},
		},
		{
			name:  "性質を持たないなら空",
			setup: func(_ w.World, _ ecs.Entity) {},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			e := world.ECS.NewEntity()
			tt.setup(world, e)

			var want []entityspec.SpecRow
			if len(tt.want) > 0 {
				// 先頭に「用途」見出し、続けて性質を1段下げた行
				want = append(want, entityspec.SpecRow{Label: query.T(world, "Uses"), Header: true})
				for _, msgid := range tt.want {
					want = append(want, entityspec.SpecRow{Label: query.T(world, msgid), Indent: 1})
				}
			}

			assert.Equal(t, want, entityspec.UseHints(world, e))
		})
	}
}
