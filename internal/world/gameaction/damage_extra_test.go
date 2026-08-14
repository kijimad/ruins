package gameaction

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyDamage_プレイヤーが関与しない場合でもpanicせず死亡処理は行われる は
// logDeathのisRelevant判定がfalseになる経路がpanicせず死亡処理自体は完了することを確認する。
func TestApplyDamage_プレイヤーが関与しない場合でもpanicせず死亡処理は行われる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 攻撃側と被弾側をどちらも実スポーンの敵にする。プレイヤーは一切関与しない
	source, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 3, Y: 3}, "bat")
	require.NoError(t, err)

	target, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 4, Y: 4}, "bat")
	require.NoError(t, err)

	// 一撃で倒せるよう被弾側のHPを低く設定する
	hp := world.Components.HP.Get(target)
	hp.Max = 10
	hp.Current = 5

	assert.NotPanics(t, func() {
		ApplyDamage(world, target, 10, source)
	})

	assert.Equal(t, 0, world.Components.HP.Get(target).Current)
	assert.True(t, world.Components.Dead.Has(target), "プレイヤーが関与しなくても死亡処理自体は行われる")
}
