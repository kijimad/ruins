package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// InvalidRangeTrigger は無効なActivationRangeを持つテスト用トリガー
type InvalidRangeTrigger struct{}

func (t InvalidRangeTrigger) Config() gc.InteractionConfig {
	return gc.InteractionConfig{
		ActivationRange: gc.ActivationRange("INVALID_RANGE"),
		ActivationWay:   gc.ActivationWayManual,
	}
}

// InvalidWayTrigger は無効なActivationWayを持つテスト用トリガー
type InvalidWayTrigger struct{}

func (t InvalidWayTrigger) Config() gc.InteractionConfig {
	return gc.InteractionConfig{
		ActivationRange: gc.ActivationRangeSameTile,
		ActivationWay:   gc.ActivationWay("INVALID_WAY"),
	}
}

// TestExecuteInteraction_NoInteractable はInteractableコンポーネントがない場合のエラーを確認
func TestExecuteInteraction_NoInteractable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	notInteractableEntity := world.Manager.NewEntity()

	_, err := ExecuteInteraction(player, notInteractableEntity, world)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Interactableを持っていません")
}

// TestExecuteInteraction_InvalidRange は無効なActivationRangeの検証エラーを確認
func TestExecuteInteraction_InvalidRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: InvalidRangeTrigger{},
	})

	_, err := ExecuteInteraction(player, triggerEntity, world)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "無効なActivationRange")
}

// TestExecuteInteraction_InvalidWay は無効なActivationWayの検証エラーを確認
func TestExecuteInteraction_InvalidWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: InvalidWayTrigger{},
	})

	_, err := ExecuteInteraction(player, triggerEntity, world)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "無効なActivationWay")
}

// TestExecuteInteraction_Door はドア相互作用の動作を確認
func TestExecuteInteraction_Door(t *testing.T) {
	t.Parallel()

	t.Run("閉じたドアを開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// ドアを作成（閉じている）
		doorEntity := world.Manager.NewEntity()
		doorEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		doorEntity.AddComponent(world.Components.Door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})
		doorEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.DoorInteraction{},
		})
		doorEntity.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
		doorEntity.AddComponent(world.Components.BlockView, &gc.BlockView{})

		// ExecuteInteractionを実行
		result, err := ExecuteInteraction(player, doorEntity, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "ドア相互作用が成功するべき")

		// ドアが開いていることを確認
		doorComp := world.Components.Door.Get(doorEntity).(*gc.Door)
		assert.True(t, doorComp.IsOpen, "ドアが開いているべき")
	})

	t.Run("開いたドアを閉じる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// ドアを作成（開いている）
		doorEntity := world.Manager.NewEntity()
		doorEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		doorEntity.AddComponent(world.Components.Door, &gc.Door{IsOpen: true, Orientation: gc.DoorOrientationHorizontal})
		doorEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.DoorInteraction{},
		})

		// ExecuteInteractionを実行
		result, err := ExecuteInteraction(player, doorEntity, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "ドア相互作用が成功するべき")

		// ドアが閉じていることを確認
		doorComp := world.Components.Door.Get(doorEntity).(*gc.Door)
		assert.False(t, doorComp.IsOpen, "ドアが閉じているべき")
	})
}

// TestExecuteInteraction_Talk は会話相互作用の動作を確認
func TestExecuteInteraction_Talk(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// NPCを作成
	npcEntity := world.Manager.NewEntity()
	npcEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
	npcEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TalkInteraction{},
	})
	npcEntity.AddComponent(world.Components.Dialog, &gc.Dialog{
		MessageKey: "test_npc_greeting",
	})
	npcEntity.AddComponent(world.Components.Name, &gc.Name{Name: "テストNPC"})
	npcEntity.AddComponent(world.Components.FactionNeutral, nil)

	// ExecuteInteractionを実行
	result, err := ExecuteInteraction(player, npcEntity, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "会話相互作用が成功するべき")
}

// TestExecuteInteraction_Item はアイテム相互作用の動作を確認
func TestExecuteInteraction_Item(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成（インベントリなし）
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// アイテムを作成
	itemEntity := world.Manager.NewEntity()
	itemEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	itemEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.ItemInteraction{},
	})
	itemEntity.AddComponent(world.Components.Name, &gc.Name{Name: "テストアイテム"})
	itemEntity.AddComponent(world.Components.Item, &gc.Item{})
	itemEntity.AddComponent(world.Components.Consumable, &gc.Consumable{})

	// ExecuteInteractionを実行（拾えるアイテムが見つからないためエラー）
	result, err := ExecuteInteraction(player, itemEntity, world)

	// 検証に失敗するためエラーになる
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestExecuteInteraction_Melee は近接攻撃相互作用の動作を確認
func TestExecuteInteraction_Melee(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成（攻撃手段なし）
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// 敵を作成
	enemyEntity := world.Manager.NewEntity()
	enemyEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
	enemyEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.MeleeInteraction{},
	})
	enemyEntity.AddComponent(world.Components.Name, &gc.Name{Name: "テスト敵"})

	// ExecuteInteractionを実行（攻撃手段がないためエラー）
	result, err := ExecuteInteraction(player, enemyEntity, world)

	// 攻撃手段がないためエラーになる
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestExecuteInteraction_Melee_BareHands は武器がない場合の素手攻撃を確認
func TestExecuteInteraction_Melee_BareHands(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成（武器なし、素手で攻撃）
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	player.AddComponent(world.Components.Attributes, &gc.Attributes{
		Strength:  gc.Attribute{Base: 5, Total: 5},
		Dexterity: gc.Attribute{Base: 5, Total: 5},
		Agility:   gc.Attribute{Base: 5, Total: 5},
		Defense:   gc.Attribute{Base: 0, Total: 0},
	})

	// 敵を作成
	enemyEntity := world.Manager.NewEntity()
	enemyEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
	enemyEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.MeleeInteraction{},
	})
	enemyEntity.AddComponent(world.Components.Name, &gc.Name{Name: "テスト敵"})
	enemyEntity.AddComponent(world.Components.Attributes, &gc.Attributes{
		Strength:  gc.Attribute{Base: 1, Total: 1},
		Dexterity: gc.Attribute{Base: 1, Total: 1},
		Agility:   gc.Attribute{Base: 1, Total: 1},
		Defense:   gc.Attribute{Base: 0, Total: 0},
	})
	enemyEntity.AddComponent(world.Components.Pools, &gc.Pools{
		HP: gc.Pool{Max: 10, Current: 10},
	})

	// 武器スロット1を選択
	world.Resources.SelectedWeaponSlot = 1

	// ExecuteInteractionを実行（素手で攻撃）
	result, err := ExecuteInteraction(player, enemyEntity, world)

	// 素手攻撃が成功すること
	require.NoError(t, err)
	require.NotNil(t, result)

	// 敵のHPが減少していることを確認
	// 攻撃には命中判定があり、稀に外れる可能性があるため、外れた場合は再試行する
	pools := world.Components.Pools.Get(enemyEntity).(*gc.Pools)
	maxRetries := 20
	for i := 0; i < maxRetries && pools.HP.Current >= 10; i++ {
		result, err = ExecuteInteraction(player, enemyEntity, world)
		require.NoError(t, err)
		require.NotNil(t, result)
	}
	assert.Less(t, pools.HP.Current, 10, "素手攻撃でダメージが入るべき")
}

// TestExecuteInteraction_Portal はポータル相互作用の動作を確認
func TestExecuteInteraction_Portal(t *testing.T) {
	t.Parallel()

	t.Run("次階への転移", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		portalEntity := world.Manager.NewEntity()
		portalEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.PortalInteraction{PortalType: gc.PortalTypeNext},
		})

		result, err := ExecuteInteraction(player, portalEntity, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "ポータル相互作用が成功するべき")
		assert.Equal(t, gc.BehaviorPortal, result.ActivityName)
	})

	t.Run("街への帰還", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		portalEntity := world.Manager.NewEntity()
		portalEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.PortalInteraction{PortalType: gc.PortalTypeTown},
		})

		result, err := ExecuteInteraction(player, portalEntity, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "帰還ポータル相互作用が成功するべき")
		assert.Equal(t, gc.BehaviorPortal, result.ActivityName)
	})

	t.Run("未知のポータルタイプ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		portalEntity := world.Manager.NewEntity()
		portalEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.PortalInteraction{PortalType: gc.PortalType("UNKNOWN")},
		})

		_, err := ExecuteInteraction(player, portalEntity, world)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "未知のポータルタイプ")
	})
}

// TestExecuteInteraction_DungeonGate はダンジョンゲート相互作用の動作を確認
func TestExecuteInteraction_DungeonGate(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})

	gateEntity := world.Manager.NewEntity()
	gateEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.DungeonGateInteraction{},
	})

	result, err := ExecuteInteraction(player, gateEntity, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "ダンジョンゲート相互作用が成功するべき")
	assert.Equal(t, gc.BehaviorDungeonGate, result.ActivityName)
}

// TestExecuteInteraction_Door_NoDoorComponent はDoorコンポーネントがない場合のエラーを確認
func TestExecuteInteraction_Door_NoDoorComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// DoorInteractionを持つがDoorコンポーネントがないエンティティ
	doorEntity := world.Manager.NewEntity()
	doorEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
	doorEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.DoorInteraction{},
	})

	_, err := ExecuteInteraction(player, doorEntity, world)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Doorコンポーネントがない")
}

// TestExecuteInteraction_Talk_NoDialogComponent はDialogコンポーネントがない場合のエラーを確認
func TestExecuteInteraction_Talk_NoDialogComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// TalkInteractionを持つがDialogコンポーネントがないエンティティ
	npcEntity := world.Manager.NewEntity()
	npcEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
	npcEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TalkInteraction{},
	})
	npcEntity.AddComponent(world.Components.Name, &gc.Name{Name: "テストNPC"})

	_, err := ExecuteInteraction(player, npcEntity, world)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Dialogコンポーネントがありません")
}

// UnknownInteraction は未知の相互作用タイプのテスト用
type UnknownInteraction struct{}

func (u UnknownInteraction) Config() gc.InteractionConfig {
	return gc.InteractionConfig{
		ActivationRange: gc.ActivationRangeSameTile,
		ActivationWay:   gc.ActivationWayManual,
	}
}

// TestExecuteInteraction_UnknownType は未知の相互作用タイプの動作を確認
func TestExecuteInteraction_UnknownType(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})

	unknownEntity := world.Manager.NewEntity()
	unknownEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: UnknownInteraction{},
	})

	result, err := ExecuteInteraction(player, unknownEntity, world)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "未知の相互作用タイプ")
}
