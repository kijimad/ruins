package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteInteraction_DungeonEnter_進入先の遺跡名を要求に載せる は、遺跡入口の相互作用が
// 入口プロップの DungeonEntrance から進入先の遺跡名を読み、WarpDungeonEnter 要求へ載せることを確認する。
// 入口ごとに進入先が違うため、名前を要求に載せて運ぶ。
func TestExecuteInteraction_DungeonEnter_進入先の遺跡名を要求に載せる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	entrance, err := lifecycle.SpawnDungeonEntrance(world, 5, 5, "森")
	require.NoError(t, err)

	_, err = ExecuteInteraction(actor, entrance, gc.InteractionDungeonEnter, world)
	require.NoError(t, err)

	req := lifecycle.ConsumeStateChange(world)
	require.NotNil(t, req, "状態変更要求が積まれる")
	payload, ok := req.Payload.(gc.WarpDungeonEnter)
	require.True(t, ok, "WarpDungeonEnter が要求される")
	assert.Equal(t, "森", payload.DefinitionName, "進入先の遺跡名が要求に載る")
}

// TestExecuteInteraction_UnknownKind は未知の種類が無効なConfigとして弾かれることを確認。
// 平坦化により未知の種類はゼロ値（無効）のConfigを返すため、発動前の検証で拒否される
func TestExecuteInteraction_UnknownKind(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	triggerEntity := world.ECS.NewEntity()
	unknown := gc.InteractionKind("UNKNOWN")
	world.Components.Interactable.Add(triggerEntity, &gc.Interactable{
		Interactions: []gc.InteractionKind{unknown},
	})

	_, err := ExecuteInteraction(player, triggerEntity, unknown, world)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ActivationRange")
}

// TestExecuteInteraction_Door は扉相互作用の動作を確認
func TestExecuteInteraction_Door(t *testing.T) {
	t.Parallel()

	t.Run("閉じた扉を開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		doorEntity, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, gc.DoorOrientationHorizontal)
		require.NoError(t, err)

		// ExecuteInteractionを実行
		result, err := ExecuteInteraction(player, doorEntity, gc.InteractionDoor, world)

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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		doorEntity, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		world.Components.Door.Get(doorEntity).IsOpen = true

		// ExecuteInteractionを実行
		result, err := ExecuteInteraction(player, doorEntity, gc.InteractionDoor, world)

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

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// 商人は Dialog と FactionNeutral を持つ
	npcEntity, err := lifecycle.SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "merchant")
	require.NoError(t, err)

	// ExecuteInteractionを実行
	result, err := ExecuteInteraction(player, npcEntity, gc.InteractionTalk, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "会話相互作用が成功するべき")
}

// TestExecuteInteraction_Item はアイテム相互作用の動作を確認
func TestExecuteInteraction_Item(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// SpawnPlayer は初期装備とバックパックを備えるため、インベントリを持たない状態を作るには
	// 素のエンティティを使う
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

	// アイテムを作成
	itemEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(itemEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
	world.Components.Interactable.Add(itemEntity, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionItem},
	})
	world.Components.Name.Add(itemEntity, &gc.Name{Name: "テストアイテム"})
	world.Components.Consumable.Add(itemEntity, &gc.Consumable{})

	result, err := ExecuteInteraction(player, itemEntity, gc.InteractionItem, world)

	require.NoError(t, err)
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
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

	enemyEntity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "fireball")
	require.NoError(t, err)

	// 攻撃能力を欠くのは不変条件違反なのでシステムエラーとして伝播する
	_, err = ExecuteInteraction(player, enemyEntity, gc.InteractionMelee, world)

	require.Error(t, err)
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
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
	world.Components.Abilities.Add(player, &gc.Abilities{
		Strength:  gc.Ability{Base: 5, Total: 5},
		Dexterity: gc.Ability{Base: 5, Total: 5},
		Agility:   gc.Ability{Base: 5, Total: 5},
		Defense:   gc.Ability{Base: 0, Total: 0},
	})

	// 敵を作成
	enemyEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(enemyEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
	world.Components.Interactable.Add(enemyEntity, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionMelee},
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
	query.GetWeaponSelection(world).Slot = 1

	result, err := ExecuteInteraction(player, enemyEntity, gc.InteractionMelee, world)
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
			Interactions: []gc.InteractionKind{gc.InteractionPortalNext},
		})

		result, err := ExecuteInteraction(player, portalEntity, gc.InteractionPortalNext, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "ポータル相互作用が成功するべき")
		assert.Equal(t, gc.BehaviorPortal, result.ActivityName)
	})

}

// TestExecuteInteraction_Door_NoDoorComponent はDoorコンポーネントがない場合のエラーを確認
func TestExecuteInteraction_Door_NoDoorComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

	// DoorInteractionを持つがDoorコンポーネントがないエンティティ
	doorEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(doorEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
	world.Components.Interactable.Add(doorEntity, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionDoor},
	})

	_, err := ExecuteInteraction(player, doorEntity, gc.InteractionDoor, world)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Door component")
}

// TestExecuteInteraction_Talk_NoDialogComponent はDialogコンポーネントがない場合のエラーを確認
func TestExecuteInteraction_Talk_NoDialogComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

	// TalkInteractionを持つがDialogコンポーネントがないエンティティ
	npcEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(npcEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
	world.Components.Interactable.Add(npcEntity, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionTalk},
	})
	world.Components.Name.Add(npcEntity, &gc.Name{Name: "テストNPC"})

	_, err := ExecuteInteraction(player, npcEntity, gc.InteractionTalk, world)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Dialog component")
}

// TestExecuteInteraction_ItemAll_座標がないとエラー はactorにGridElementがない場合、
// 全アイテム拾得が座標未設定のエラーになることを確認する
func TestExecuteInteraction_ItemAll_座標がないとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// GridElementを持たないactor
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	triggerEntity := world.ECS.NewEntity()

	_, err := ExecuteInteraction(player, triggerEntity, gc.InteractionItemAll, world)

	require.ErrorIs(t, err, ErrPositionNotFound)
}

// TestExecuteInteraction_ItemAll_同じタイルのアイテムを拾う は同一タイル上のアイテムが
// まとめて拾得されることを確認する
func TestExecuteInteraction_ItemAll_同じタイルのアイテムを拾う(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	item, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 1)
	require.NoError(t, err)

	triggerEntity := world.ECS.NewEntity()
	result, err := ExecuteInteraction(player, triggerEntity, gc.InteractionItemAll, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "足元のアイテム拾得が成功するべき")
	assert.True(t, world.Components.LocationInBackpack.Has(item), "アイテムがバックパックへ移るべき")
}

// TestExecuteInteraction_Storage は収納相互作用がストレージメニューの状態変更要求を積むことを確認する
func TestExecuteInteraction_Storage(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})

	storageEntity := world.ECS.NewEntity()
	world.Components.Interactable.Add(storageEntity, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionStorage},
	})

	result, err := ExecuteInteraction(player, storageEntity, gc.InteractionStorage, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "収納相互作用が成功するべき")
	assert.Equal(t, gc.BehaviorStorage, result.ActivityName)

	req := lifecycle.ConsumeStateChange(world)
	require.NotNil(t, req, "状態変更要求が積まれる")
	payload, ok := req.Payload.(gc.OpenStorage)
	require.True(t, ok, "OpenStorageが要求される")
	assert.Equal(t, storageEntity, payload.StorageEntity, "対象の収納エンティティが要求に載る")
}

// TestExecuteInteraction_Disassemble_工具がなければ何もしない は分解相互作用の起動をラップする
// executeDisassemble が実際にDisassembleBehaviorへ委譲することを確認する。工具がないので
// Validate がUserErrorを返し、Executeがそれをgamelogへ出してSuccess=falseで閉じる
func TestExecuteInteraction_Disassemble_工具がなければ何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player := newDisassembleTestPlayer(world)

	crate, err := lifecycle.SpawnProp(world, "crate", 11, 10)
	require.NoError(t, err)

	result, err := ExecuteInteraction(player, crate, gc.InteractionDisassemble, world)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success, "工具がないので分解は開始できないべき")
	assert.Equal(t, gc.BehaviorDisassemble, result.ActivityName)
}

// TestExecuteInteraction_Fixed は固定物へのMeleeInteractionの動作を確認する
func TestExecuteInteraction_Fixed(t *testing.T) {
	t.Parallel()

	t.Run("Propを攻撃できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Abilities.Add(player, &gc.Abilities{
			Strength:  gc.Ability{Base: 5, Total: 5},
			Dexterity: gc.Ability{Base: 5, Total: 5},
		})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Name.Add(prop, &gc.Name{Name: "木箱"})
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 30})
		world.Components.Interactable.Add(prop, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionMelee},
		})

		query.GetWeaponSelection(world).Slot = 1

		result, err := ExecuteInteraction(player, prop, gc.InteractionMelee, world)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, gc.BehaviorMelee, result.ActivityName)

		hp := world.Components.HP.Get(prop)
		assert.Less(t, hp.Current, 30, "攻撃でダメージが入るべき")
	})

	t.Run("Dead済みのPropは攻撃できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Abilities.Add(player, &gc.Abilities{
			Strength:  gc.Ability{Base: 5, Total: 5},
			Dexterity: gc.Ability{Base: 5, Total: 5},
		})

		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Name.Add(prop, &gc.Name{Name: "壊れた木箱"})
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 0})
		world.Components.Dead.Add(prop, &gc.Dead{})
		world.Components.Interactable.Add(prop, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionMelee},
		})

		// 対象選択が死亡を除外するため、Dead 対象への攻撃は不変条件違反でシステムエラーになる
		_, err := ExecuteInteraction(player, prop, gc.InteractionMelee, world)

		require.Error(t, err)
	})
}
