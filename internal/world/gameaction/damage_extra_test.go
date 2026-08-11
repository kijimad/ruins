package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// TestApplyDamage_プレイヤーが関与しない場合でもpanicせず死亡処理は行われる は
// logDeathのisRelevant判定がfalseになる経路がpanicせず死亡処理自体は完了することを確認する。
func TestApplyDamage_プレイヤーが関与しない場合でもpanicせず死亡処理は行われる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	source := world.ECS.NewEntity()
	world.Components.Name.Add(source, &gc.Name{Name: "コウモリA"})

	target := world.ECS.NewEntity()
	world.Components.Name.Add(target, &gc.Name{Name: "コウモリB"})
	world.Components.HP.Add(target, &gc.HP{Max: 10, Current: 5})

	assert.NotPanics(t, func() {
		ApplyDamage(world, target, 10, source)
	})

	hp := world.Components.HP.Get(target)
	assert.Equal(t, 0, hp.Current)
	assert.True(t, world.Components.Dead.Has(target), "プレイヤーが関与しなくても死亡処理自体は行われる")
}
