package save

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSerdeWholeWorldRoundtrip は代表的なエンティティ（プレイヤー・敵・アイテム・扉・置物）を
// 配置し、production の SaveWorld/LoadWorld 経由で丸ごと保存・復元されることを検証する。
func TestSerdeWholeWorldRoundtrip(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	_, err = lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 8, Y: 8}, "火の玉")
	require.NoError(t, err)
	// 回復薬は平坦化された ProvidesHealing を持つ。serde 往復で保存されることを確認する
	_, err = lifecycle.SpawnFieldItem(world, "回復薬", 3, 3, 2)
	require.NoError(t, err)
	_, err = lifecycle.SpawnDoor(world, 4, 4, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "木箱", 6, 6)
	require.NoError(t, err)

	require.NoError(t, manager.SaveWorld(world, "roundtrip"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "roundtrip"))

	// プレイヤーが復元される
	playerCount := 0
	pq := ecs.NewFilter1[gc.Player](newWorld.ECS).Query()
	for pq.Next() {
		playerCount++
	}
	assert.Equal(t, 1, playerCount, "プレイヤーが1体復元される")

	// 敵（丸ごと保存のため復元される）
	enemyCount := 0
	eq := ecs.NewFilter1[gc.SoloAI](newWorld.ECS).Query()
	for eq.Next() {
		enemyCount++
	}
	assert.GreaterOrEqual(t, enemyCount, 1, "敵が復元される")

	// 扉（地形も丸ごと保存される）。Interactableも復元され、復帰後に操作できる
	doorCount := 0
	doorHasInteraction := false
	dq := ecs.NewFilter1[gc.Door](newWorld.ECS).Query()
	for dq.Next() {
		doorCount++
		e := dq.Entity()
		if newWorld.Components.Interactable.Has(e) {
			for _, in := range newWorld.Components.Interactable.Get(e).Interactions {
				if in == gc.InteractionDoor {
					doorHasInteraction = true
				}
			}
		}
	}
	assert.Equal(t, 1, doorCount, "扉が復元される")
	assert.True(t, doorHasInteraction, "扉のInteractionが復元され、復帰後に開閉できる")

	// ProvidesHealing が復元される（平坦化により serde 可能になった）
	healingFound := false
	hq := ecs.NewFilter1[gc.ProvidesHealing](newWorld.ECS).Query()
	for hq.Next() {
		healingFound = true
	}
	assert.True(t, healingFound, "回復効果コンポーネントが復元される")
}

// TestSerde_DungeonLocationPersists は現在地（Dungeonの階層・定義名）が保存・復元されることを検証する。
// ロード時の復帰先ステート再構築はこの2フィールドに依存する。
func TestSerde_DungeonLocationPersists(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	_, err = lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	dungeonState := query.GetDungeon(world)
	dungeonState.Depth = 3
	dungeonState.DefinitionName = "ダンジョン"

	require.NoError(t, manager.SaveWorld(world, "location"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "location"))

	restored := query.GetDungeon(newWorld)
	require.NotNil(t, restored)
	assert.Equal(t, 3, restored.Depth, "階層が復元される")
	assert.Equal(t, "ダンジョン", restored.DefinitionName, "ダンジョン定義名が復元される")

	// 視界マップは json:"-" で除外されるが、reestablishSingleton が空mapで初期化する。
	// nilのままだと視界処理で書き込み時にpanicするため非nilであること
	assert.NotNil(t, restored.ExploredTiles, "探索済みマップが空mapで初期化される")
	assert.NotNil(t, restored.VisibleTiles, "可視マップが空mapで初期化される")
}

// TestSerde_StageBoundとSuspendedが往復する は共存方式の永続状態が
// セーブ・ロードで復元されることを検証する。退避中ステージのエンティティは StageBound と
// Suspended を持ったまま world に残るため、これらが serde 対象で、ロード後も稼働/非稼働の
// 別が保たれる必要がある。CurrentStage も含めて往復を確認する。
func TestSerde_StageBoundとSuspendedが往復する(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	_, err = lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	key := gc.NewDungeonStage(2)
	query.GetDungeon(world).CurrentStage = key

	// 退避中ステージのエンティティ相当。StageBound を持ち Suspended で退避されている
	e := world.ECS.NewEntity()
	world.Components.StageBound.Add(e, &gc.StageBound{Key: key})
	world.Components.Suspended.Add(e, &gc.Suspended{})

	require.NoError(t, manager.SaveWorld(world, "stage"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "stage"))

	assert.Equal(t, key, query.GetDungeon(newWorld).CurrentStage, "現ステージが復元される")

	var restored []ecs.Entity
	q := ecs.NewFilter1[gc.StageBound](newWorld.ECS).Query()
	for q.Next() {
		restored = append(restored, q.Entity())
	}
	require.Len(t, restored, 1, "StageBound を持つエンティティが1つ復元される")
	assert.Equal(t, key, newWorld.Components.StageBound.Get(restored[0]).Key, "所属ステージが復元される")
	assert.True(t, newWorld.Components.Suspended.Has(restored[0]), "退避状態が復元される")
}

// TestSerde_SoloAITargetEntityRemaps は SoloAI.TargetEntity（*ecs.Entity）が
// セーブ・ロード往復でエンティティ参照として整合することを検証する。
// ark-serde はエンティティプール（ID・世代）ごと保存・再構築し、参照を再マッピングするため、
// 保存前後でエンティティIDが変わっても相互参照の整合性が保たれる。
// 戦闘中（TargetEntity が非nil）のセーブ→ロードで参照が壊れないことを保証する。
func TestSerde_SoloAITargetEntityRemaps(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 8, Y: 8}, "火の玉")
	require.NoError(t, err)

	// 敵のSoloAIがプレイヤーを標的にしている戦闘中状態を作る
	require.True(t, world.Components.SoloAI.Has(enemy), "敵がSoloAIを持つ")
	world.Components.SoloAI.Get(enemy).TargetEntity = &player

	require.NoError(t, manager.SaveWorld(world, "aitarget"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "aitarget"))

	// 復元後のプレイヤーと敵を特定する
	var restoredPlayer ecs.Entity
	pq := ecs.NewFilter1[gc.Player](newWorld.ECS).Query()
	for pq.Next() {
		restoredPlayer = pq.Entity()
	}
	var restoredEnemy ecs.Entity
	eq := ecs.NewFilter1[gc.SoloAI](newWorld.ECS).Query()
	for eq.Next() {
		restoredEnemy = eq.Entity()
	}

	// TargetEntity が復元され、生存する復元後プレイヤーを指す
	ai := newWorld.Components.SoloAI.Get(restoredEnemy)
	require.NotNil(t, ai.TargetEntity, "TargetEntityが復元される")
	require.True(t, newWorld.ECS.Alive(*ai.TargetEntity), "TargetEntityが生存エンティティを指す")
	assert.Equal(t, restoredPlayer, *ai.TargetEntity, "TargetEntityが復元後プレイヤーへ整合する")
}
