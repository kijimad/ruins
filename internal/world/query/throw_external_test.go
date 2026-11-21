package query_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThrowAttackOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		item   string
		want   gc.ThrowAttack
		wantOK bool
	}{
		{name: "投げの欄を書いた物は欄の値", item: "sharp_stone", want: gc.ThrowAttack{Damage: 7, Accuracy: 80}, wantOK: true},
		{name: "手榴弾は欄の値と爆風", item: "grenade", want: gc.ThrowAttack{Damage: 20, Accuracy: 75, Blast: &gc.Blast{Radius: 1, Element: rawschema.ElementNONE}}, wantOK: true},
		{name: "火炎手榴弾の爆風は火の属性", item: "flame_grenade", want: gc.ThrowAttack{Damage: 25, Accuracy: 75, Blast: &gc.Blast{Radius: 1, Element: rawschema.ElementFIRE}}, wantOK: true},
		{name: "毒手榴弾の爆風は毒を付ける", item: "poison_grenade", want: gc.ThrowAttack{Damage: 10, Accuracy: 75, Blast: &gc.Blast{Radius: 1, Element: rawschema.ElementNONE, Condition: "venom"}}, wantOK: true},
		{name: "欄の無い物は重さから導く", item: "bread", want: gc.ThrowAttack{Damage: 1, Accuracy: 68}, wantOK: true},
		{name: "5 kg を超える物は投げられない", item: "deployable_bed", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
			require.NoError(t, err)
			item, err := lifecycle.SpawnBackpackItem(world, tt.item, 1)
			require.NoError(t, err)

			got, ok := query.ThrowAttackOf(world, item)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
