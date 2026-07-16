package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecute_アクターのみ処理し他エンティティを進めない は、Execute の即時アクション処理が
// ProcessTurnForEntity 経由でアクターのみを対象とし、他エンティティの継続アクティビティを
// 進めないことを検証する回帰テスト。
//
// 旧実装は Execute 内で ProcessTurn(world)（全エンティティ処理）を呼んでおり、入れ子呼び出し時に
// 処理中のアクティビティ・コンポーネントが別処理で削除・再利用され、ダングリング参照で panic した
// （攻撃時に Finish で comp.Target が nil になるなど）。本テストは「他エンティティが進まない」ことで
// アクターのみ処理される契約を固定する。
func TestExecute_アクターのみ処理し他エンティティを進めない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)
	actor, err := lifecycle.SpawnEnemy(world, 12, 10, "火の玉")
	require.NoError(t, err)
	other, err := lifecycle.SpawnEnemy(world, 30, 30, "火の玉")
	require.NoError(t, err)

	// other に継続アクティビティを手動で持たせる（Running・残り5ターン）
	otherComp := &gc.Activity{
		BehaviorName: gc.BehaviorRest,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   5,
		TurnsLeft:    5,
	}
	require.NoError(t, query.SetActivity(world, other, otherComp))

	// actor が即時アクション（待機）を実行する
	_, err = Execute(&WaitActivity{Duration: 1, Reason: "テスト"}, actor, world)
	require.NoError(t, err)

	// other のアクティビティは進んでいない（Execute はアクターのみ処理する）
	after := query.GetActivity(world, other)
	require.NotNil(t, after, "他エンティティのアクティビティは残っている")
	assert.Equal(t, 5, after.TurnsLeft,
		"Execute はアクターのみ処理し、他エンティティの継続アクティビティを進めない")
}
