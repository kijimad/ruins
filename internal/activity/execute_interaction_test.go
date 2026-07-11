package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteInteraction_UnknownKind は未知の種類が無効なConfigとして弾かれることを確認。
// 平坦化により未知の種類はゼロ値（無効）のConfigを返すため、発動前の検証で拒否される
func TestExecuteInteraction_UnknownKind(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	triggerEntity := world.ECS.NewEntity()
	unknown := gc.InteractionData{Kind: "UNKNOWN"}
	world.Components.Interactable.Add(triggerEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{unknown},
	})

	_, err := ExecuteInteraction(player, triggerEntity, unknown, world)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "無効なActivationRange")
}

// TestExecuteInteraction_Door は扉相互作用の動作を確認
func TestExecuteInteraction_Door(t *testing.T) {
	t.Parallel()

	t.Run("閉じた扉を開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

		// 扉を作成（閉じている）
		doorEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(doorEntity, &gc.GridElement{X: 11, Y: 10})
		world.Components.Door.Add(doorEntity, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})
		world.Components.Interactable.Add(doorEntity, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionDoor}},
		})
		world.Components.BlockPass.Add(doorEntity, &gc.BlockPass{})
		world.Components.BlockView.Add(doorEntity, &gc.BlockView{})

		// ExecuteInteractionを実行
		result, err := ExecuteInteraction(player, doorEntity, gc.InteractionData{Kind: gc.InteractionDoor}, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "扉相互作用が成功するべき")

		// 扉が開いていることを確認
		doorComp := world.Components.Door.Get(doorEntity)
		assert.True(t, doorComp.IsOpen, "扉が開いているべき")
	})

	t.Run("開いた扉を閉じる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

		// 扉を作成（開いている）
		doorEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(doorEntity, &gc.GridElement{X: 11, Y: 10})
		world.Components.Door.Add(doorEntity, &gc.Door{IsOpen: true, Orientation: gc.DoorOrientationHorizontal})
		world.Components.Interactable.Add(doorEntity, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionDoor}},
		})

		// ExecuteInteractionを実行
		result, err := ExecuteInteraction(player, doorEntity, gc.InteractionData{Kind: gc.InteractionDoor}, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "扉相互作用が成功するべき")

		// 扉が閉じていることを確認
		doorComp := world.Components.Door.Get(doorEntity)
		assert.False(t, doorComp.IsOpen, "扉が閉じているべき")
	})
}

// TestExecuteInteraction_Talk は会話相互作用の動作を確認
func TestExecuteInteraction_Talk(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.TurnBased.Add(player, &gc.TurnBased{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

	// NPCを作成
	npcEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(npcEntity, &gc.GridElement{X: 11, Y: 10})
	world.Components.Interactable.Add(npcEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{{Kind: gc.InteractionTalk}},
	})
	world.Components.Dialog.Add(npcEntity, &gc.Dialog{
		MessageKey: "test_npc_greeting",
	})
	world.Components.Name.Add(npcEntity, &gc.Name{Name: "テストNPC"})
	world.Components.FactionNeutral.Add(npcEntity, &gc.FactionNeutralData{})

	// ExecuteInteractionを実行
	result, err := ExecuteInteraction(player, npcEntity, gc.InteractionData{Kind: gc.InteractionTalk}, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "会話相互作用が成功するべき")
}

// TestExecuteInteraction_Item はアイテム相互作用の動作を確認
func TestExecuteInteraction_Item(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成（インベントリなし）
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

	// アイテムを作成
	itemEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(itemEntity, &gc.GridElement{X: 10, Y: 10})
	world.Components.Interactable.Add(itemEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{{Kind: gc.InteractionItem}},
	})
	world.Components.Name.Add(itemEntity, &gc.Name{Name: "テストアイテム"})
	world.Components.Consumable.Add(itemEntity, &gc.Consumable{})

	// ExecuteInteractionを実行（拾えるアイテムが見つからないためエラー）
	result, err := ExecuteInteraction(player, itemEntity, gc.InteractionData{Kind: gc.InteractionItem}, world)

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
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

	// 敵を作成
	enemyEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(enemyEntity, &gc.GridElement{X: 11, Y: 10})
	world.Components.Interactable.Add(enemyEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{{Kind: gc.InteractionMelee}},
	})
	world.Components.Name.Add(enemyEntity, &gc.Name{Name: "テスト敵"})

	// ExecuteInteractionを実行（攻撃手段がないためエラー）
	result, err := ExecuteInteraction(player, enemyEntity, gc.InteractionData{Kind: gc.InteractionMelee}, world)

	// 攻撃手段がないためエラーになる
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestExecuteInteraction_Melee_BareHands は武器がない場合の素手攻撃を確認
func TestExecuteInteraction_Melee_BareHands(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(42, 0))

	// プレイヤーを作成（武器なし、素手で攻撃）
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.TurnBased.Add(player, &gc.TurnBased{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})
	world.Components.Abilities.Add(player, &gc.Abilities{
		Strength:  gc.Ability{Base: 5, Total: 5},
		Dexterity: gc.Ability{Base: 5, Total: 5},
		Agility:   gc.Ability{Base: 5, Total: 5},
		Defense:   gc.Ability{Base: 0, Total: 0},
	})

	// 敵を作成
	enemyEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(enemyEntity, &gc.GridElement{X: 11, Y: 10})
	world.Components.Interactable.Add(enemyEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{{Kind: gc.InteractionMelee}},
	})
	world.Components.Name.Add(enemyEntity, &gc.Name{Name: "テスト敵"})
	world.Components.Abilities.Add(enemyEntity, &gc.Abilities{
		Strength:  gc.Ability{Base: 1, Total: 1},
		Dexterity: gc.Ability{Base: 1, Total: 1},
		Agility:   gc.Ability{Base: 1, Total: 1},
		Defense:   gc.Ability{Base: 0, Total: 0},
	})
	world.Components.HP.Add(enemyEntity, &gc.HP{Max: 10, Current: 10})

	// 武器スロット1を選択
	query.GetDungeon(world).SelectedWeaponSlot = 1

	result, err := ExecuteInteraction(player, enemyEntity, gc.InteractionData{Kind: gc.InteractionMelee}, world)
	require.NoError(t, err)
	require.NotNil(t, result)

	hp := world.Components.HP.Get(enemyEntity)
	assert.Less(t, hp.Current, 10, "素手攻撃でダメージが入るべき")
}

// TestExecuteInteraction_Portal はポータル相互作用の動作を確認
func TestExecuteInteraction_Portal(t *testing.T) {
	t.Parallel()

	t.Run("次階への転移", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		portalEntity := world.ECS.NewEntity()
		world.Components.Interactable.Add(portalEntity, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionPortal, PortalType: gc.PortalTypeNext}},
		})

		result, err := ExecuteInteraction(player, portalEntity, gc.InteractionData{Kind: gc.InteractionPortal, PortalType: gc.PortalTypeNext}, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "ポータル相互作用が成功するべき")
		assert.Equal(t, gc.BehaviorPortal, result.ActivityName)
	})

	t.Run("街への帰還", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		portalEntity := world.ECS.NewEntity()
		world.Components.Interactable.Add(portalEntity, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionPortal, PortalType: gc.PortalTypeTown}},
		})

		result, err := ExecuteInteraction(player, portalEntity, gc.InteractionData{Kind: gc.InteractionPortal, PortalType: gc.PortalTypeTown}, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "帰還ポータル相互作用が成功するべき")
		assert.Equal(t, gc.BehaviorPortal, result.ActivityName)
	})

	t.Run("未知のポータルタイプ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		portalEntity := world.ECS.NewEntity()
		world.Components.Interactable.Add(portalEntity, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionPortal, PortalType: gc.PortalType("UNKNOWN")}},
		})

		_, err := ExecuteInteraction(player, portalEntity, gc.InteractionData{Kind: gc.InteractionPortal, PortalType: gc.PortalType("UNKNOWN")}, world)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "未知のポータルタイプ")
	})
}

// TestExecuteInteraction_DungeonGate はダンジョンゲート相互作用の動作を確認
func TestExecuteInteraction_DungeonGate(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})

	gateEntity := world.ECS.NewEntity()
	world.Components.Interactable.Add(gateEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{{Kind: gc.InteractionDungeonGate}},
	})

	result, err := ExecuteInteraction(player, gateEntity, gc.InteractionData{Kind: gc.InteractionDungeonGate}, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "ダンジョンゲート相互作用が成功するべき")
	assert.Equal(t, gc.BehaviorDungeonGate, result.ActivityName)
}

// TestExecuteInteraction_DoorLock はドアロック相互作用の動作を確認
func TestExecuteInteraction_DoorLock(t *testing.T) {
	t.Parallel()

	t.Run("全扉をロックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

		// 扉を2つ作成
		door1 := world.ECS.NewEntity()
		world.Components.Door.Add(door1, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})
		door2 := world.ECS.NewEntity()
		world.Components.Door.Add(door2, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationVertical})

		// DoorLockTriggerエンティティを作成
		trigger := world.ECS.NewEntity()
		world.Components.GridElement.Add(trigger, &gc.GridElement{X: 10, Y: 10})
		world.Components.Interactable.Add(trigger, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionDoorLock}},
		})

		result, err := ExecuteInteraction(player, trigger, gc.InteractionData{Kind: gc.InteractionDoorLock}, world)
		require.NoError(t, err)
		assert.True(t, result.Success)

		// 全扉がロックされていることを確認
		doorComp1 := world.Components.Door.Get(door1)
		doorComp2 := world.Components.Door.Get(door2)
		assert.True(t, doorComp1.Locked, "扉1がロックされるべき")
		assert.True(t, doorComp2.Locked, "扉2がロックされるべき")
	})

	t.Run("既にロック済みの扉はスキップする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		// 既ロックの扉
		door := world.ECS.NewEntity()
		world.Components.Door.Add(door, &gc.Door{IsOpen: false, Locked: true})

		trigger := world.ECS.NewEntity()
		world.Components.Interactable.Add(trigger, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionDoorLock}},
		})

		result, err := ExecuteInteraction(player, trigger, gc.InteractionData{Kind: gc.InteractionDoorLock}, world)
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("開いている扉を閉じてからロックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		// 開いた扉を作成
		door := world.ECS.NewEntity()
		world.Components.Door.Add(door, &gc.Door{IsOpen: true, Orientation: gc.DoorOrientationHorizontal})
		world.Components.SpriteRender.Add(door, &gc.SpriteRender{SpriteSheetName: "field", SpriteKey: "door_horizontal_open"})

		trigger := world.ECS.NewEntity()
		world.Components.Interactable.Add(trigger, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionDoorLock}},
		})

		result, err := ExecuteInteraction(player, trigger, gc.InteractionData{Kind: gc.InteractionDoorLock}, world)
		require.NoError(t, err)
		assert.True(t, result.Success)

		doorComp := world.Components.Door.Get(door)
		assert.False(t, doorComp.IsOpen, "扉が閉じられるべき")
		assert.True(t, doorComp.Locked, "扉がロックされるべき")
	})
}

// TestExecuteInteraction_Door_Locked はロック済み扉の相互作用を確認
func TestExecuteInteraction_Door_Locked(t *testing.T) {
	t.Parallel()

	t.Run("ロックされた扉は開けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

		doorEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(doorEntity, &gc.GridElement{X: 11, Y: 10})
		world.Components.Door.Add(doorEntity, &gc.Door{IsOpen: false, Locked: true, Orientation: gc.DoorOrientationHorizontal})
		world.Components.Interactable.Add(doorEntity, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionDoor}},
		})
		world.Components.BlockPass.Add(doorEntity, &gc.BlockPass{})
		world.Components.BlockView.Add(doorEntity, &gc.BlockView{})

		result, err := ExecuteInteraction(player, doorEntity, gc.InteractionData{Kind: gc.InteractionDoor}, world)
		require.NoError(t, err)
		require.NotNil(t, result)

		// ロック済み扉は開かない
		doorComp := world.Components.Door.Get(doorEntity)
		assert.False(t, doorComp.IsOpen, "ロックされた扉は開かないべき")
	})
}

// TestExecuteInteraction_Door_NoDoorComponent はDoorコンポーネントがない場合のエラーを確認
func TestExecuteInteraction_Door_NoDoorComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

	// DoorInteractionを持つがDoorコンポーネントがないエンティティ
	doorEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(doorEntity, &gc.GridElement{X: 11, Y: 10})
	world.Components.Interactable.Add(doorEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{{Kind: gc.InteractionDoor}},
	})

	_, err := ExecuteInteraction(player, doorEntity, gc.InteractionData{Kind: gc.InteractionDoor}, world)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Doorコンポーネントがない")
}

// TestExecuteInteraction_Talk_NoDialogComponent はDialogコンポーネントがない場合のエラーを確認
func TestExecuteInteraction_Talk_NoDialogComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})

	// TalkInteractionを持つがDialogコンポーネントがないエンティティ
	npcEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(npcEntity, &gc.GridElement{X: 11, Y: 10})
	world.Components.Interactable.Add(npcEntity, &gc.Interactable{
		Interactions: []gc.InteractionData{{Kind: gc.InteractionTalk}},
	})
	world.Components.Name.Add(npcEntity, &gc.Name{Name: "テストNPC"})

	_, err := ExecuteInteraction(player, npcEntity, gc.InteractionData{Kind: gc.InteractionTalk}, world)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Dialogコンポーネントがありません")
}

// TestExecuteInteraction_Prop はPropへのMeleeInteractionの動作を確認する
func TestExecuteInteraction_Prop(t *testing.T) {
	t.Parallel()

	t.Run("Propを攻撃できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})
		world.Components.Abilities.Add(player, &gc.Abilities{
			Strength:  gc.Ability{Base: 5, Total: 5},
			Dexterity: gc.Ability{Base: 5, Total: 5},
		})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{X: 11, Y: 10})
		world.Components.Name.Add(prop, &gc.Name{Name: "木箱"})
		world.Components.Prop.Add(prop, &gc.Prop{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 30})
		world.Components.Interactable.Add(prop, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionMelee}},
		})

		query.GetDungeon(world).SelectedWeaponSlot = 1

		result, err := ExecuteInteraction(player, prop, gc.InteractionData{Kind: gc.InteractionMelee}, world)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, gc.BehaviorAttack, result.ActivityName)

		hp := world.Components.HP.Get(prop)
		assert.Less(t, hp.Current, 30, "攻撃でダメージが入るべき")
	})

	t.Run("Dead済みのPropは攻撃できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})
		world.Components.Abilities.Add(player, &gc.Abilities{
			Strength:  gc.Ability{Base: 5, Total: 5},
			Dexterity: gc.Ability{Base: 5, Total: 5},
		})

		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{X: 11, Y: 10})
		world.Components.Name.Add(prop, &gc.Name{Name: "壊れた木箱"})
		world.Components.Prop.Add(prop, &gc.Prop{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 0})
		world.Components.Dead.Add(prop, &gc.Dead{})
		world.Components.Interactable.Add(prop, &gc.Interactable{
			Interactions: []gc.InteractionData{{Kind: gc.InteractionMelee}},
		})

		result, err := ExecuteInteraction(player, prop, gc.InteractionData{Kind: gc.InteractionMelee}, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success)
	})
}
