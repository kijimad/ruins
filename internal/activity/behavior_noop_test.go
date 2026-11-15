package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBehavior_NoOpStep_ゲームログに出力しない は、log.Debug のみで常に nil を返す
// Behavior の Start/Finish/Canceled をまとめて検証する。これらは実装上固定で失敗しようがないため、
// 検証できる唯一の契約は「ゲームログに何も出さない」こと。将来うっかり出力を足したときに捕まえる
func TestBehavior_NoOpStep_ゲームログに出力しない(t *testing.T) {
	t.Parallel()

	type step func(comp *gc.Activity, actor ecs.Entity, world w.World) error

	tests := []struct {
		name string
		fn   step
	}{
		{"MeleeBehavior.Canceled", (&MeleeBehavior{}).Canceled},
		{"OpenDoorBehavior.Canceled", (&OpenDoorBehavior{}).Canceled},
		{"CloseDoorBehavior.Canceled", (&CloseDoorBehavior{}).Canceled},
		{"DropBehavior.Start", (&DropBehavior{}).Start},
		{"DropBehavior.Finish", (&DropBehavior{}).Finish},
		{"DropBehavior.Canceled", (&DropBehavior{}).Canceled},
		{"MoveBehavior.Canceled", (&MoveBehavior{}).Canceled},
		{"PickupBehavior.Start", (&PickupBehavior{}).Start},
		{"PickupBehavior.Finish", (&PickupBehavior{}).Finish},
		{"PickupBehavior.Canceled", (&PickupBehavior{}).Canceled},
		{"ShootBehavior.Canceled", (&ShootBehavior{}).Canceled},
		{"SwapBehavior.Canceled", (&SwapBehavior{}).Canceled},
		{"TalkBehavior.Canceled", (&TalkBehavior{}).Canceled},
		{"UseItemBehavior.Canceled", (&UseItemBehavior{}).Canceled},
		{"ReloadBehavior.Finish", (&ReloadBehavior{}).Finish},
		{"RestBehavior.Start", (&RestBehavior{}).Start},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)

			comp := &gc.Activity{CancelReason: "テスト中断"}
			require.NoError(t, tt.fn(comp, player, world))
			assert.Empty(t, query.GetGameLog(world).GetRecent(1))
		})
	}
}
