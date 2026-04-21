package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestSetMaxHPSP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		vitality    int
		strength    int
		sensation   int
		dexterity   int
		agility     int
		expectedHP  int
		expectedSP  int
		description string
	}{
		{
			name:        "基本ステータス",
			vitality:    10,
			strength:    8,
			sensation:   7,
			dexterity:   6,
			agility:     9,
			expectedHP:  30 + 10*8 + 8 + 7, // 30 + 95 = 125
			expectedSP:  10*2 + 6 + 9,      // 35
			description: "基本的なHP/SP計算",
		},
		{
			name:        "中ステータス",
			vitality:    15,
			strength:    12,
			sensation:   10,
			dexterity:   8,
			agility:     11,
			expectedHP:  30 + 15*8 + 12 + 10, // 30 + 142 = 172
			expectedSP:  15*2 + 8 + 11,       // 49
			description: "中ステータスでのHP/SP計算",
		},
		{
			name:        "高ステータス",
			vitality:    20,
			strength:    18,
			sensation:   15,
			dexterity:   14,
			agility:     16,
			expectedHP:  30 + 20*8 + 18 + 15, // 30 + 193 = 223
			expectedSP:  20*2 + 14 + 16,      // 70
			description: "高ステータスでのHP/SP計算",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// 独立したworldを作成
			world := testutil.InitTestWorld(t)

			// エンティティを作成
			entity := world.Manager.NewEntity()

			// Abilitiesコンポーネントを追加（BaseとTotalを0に設定してsetMaxHPSPの初期化をテスト）
			entity.AddComponent(world.Components.Abilities, &gc.Abilities{
				Vitality:  gc.Ability{Base: tt.vitality, Total: 0},
				Strength:  gc.Ability{Base: tt.strength, Total: 0},
				Sensation: gc.Ability{Base: tt.sensation, Total: 0},
				Dexterity: gc.Ability{Base: tt.dexterity, Total: 0},
				Agility:   gc.Ability{Base: tt.agility, Total: 0},
				Defense:   gc.Ability{Base: 5, Total: 0},
			})

			// Poolsコンポーネントを追加
			entity.AddComponent(world.Components.Pools, &gc.Pools{
				HP: gc.Pool{Current: 0, Max: 0},
				SP: gc.Pool{Current: 0, Max: 0},
			})

			// 関数を実行
			err := setMaxHPSP(world, entity)
			require.NoError(t, err)

			// 結果を検証
			pools := world.Components.Pools.Get(entity).(*gc.Pools)
			abils := world.Components.Abilities.Get(entity).(*gc.Abilities)

			// Totalが正しく初期化されたことを確認
			assert.Equal(t, tt.vitality, abils.Vitality.Total, "体力のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.strength, abils.Strength.Total, "力のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.sensation, abils.Sensation.Total, "感覚のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.dexterity, abils.Dexterity.Total, "器用さのTotal値が正しく初期化されていない")
			assert.Equal(t, tt.agility, abils.Agility.Total, "素早さのTotal値が正しく初期化されていない")

			// HP/SPが正しく計算されたことを確認
			assert.Equal(t, tt.expectedHP, pools.HP.Max, "最大HPの計算が正しくない: %s", tt.description)
			assert.Equal(t, tt.expectedHP, pools.HP.Current, "現在HPが最大HPと同じでない: %s", tt.description)
			assert.Equal(t, tt.expectedSP, pools.SP.Max, "最大SPの計算が正しくない: %s", tt.description)
			assert.Equal(t, tt.expectedSP, pools.SP.Current, "現在SPが最大SPと同じでない: %s", tt.description)

			// クリーンアップ
			world.Manager.DeleteEntity(entity)
		})
	}
}

func TestSetMaxHPSP_WithoutComponents(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 必要なコンポーネントがないエンティティ
	entity := world.Manager.NewEntity()

	// 関数を実行してエラーが発生することを確認
	err := setMaxHPSP(world, entity)
	require.Error(t, err, "必要なコンポーネントがない場合はエラーを返すべき")
	assert.Contains(t, err.Error(), "does not have required components", "エラーメッセージが適切であるべき")

	// クリーンアップ
	world.Manager.DeleteEntity(entity)
}

func TestFullRecover(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用エンティティを作成
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.Abilities, &gc.Abilities{
		Vitality:  gc.Ability{Base: 10, Total: 0},
		Strength:  gc.Ability{Base: 8, Total: 0},
		Sensation: gc.Ability{Base: 7, Total: 0},
		Dexterity: gc.Ability{Base: 6, Total: 0},
		Agility:   gc.Ability{Base: 9, Total: 0},
		Defense:   gc.Ability{Base: 5, Total: 0},
	})
	entity.AddComponent(world.Components.Pools, &gc.Pools{
		HP: gc.Pool{Current: 0, Max: 0},
		SP: gc.Pool{Current: 0, Max: 0},
	})

	// FullRecoverを実行
	err := FullRecover(world, entity)
	require.NoError(t, err, "FullRecoverがエラーを返すべきではない")

	// 結果を検証
	pools := world.Components.Pools.Get(entity).(*gc.Pools)
	abils := world.Components.Abilities.Get(entity).(*gc.Abilities)

	// 能力値のTotalが正しく設定されたことを確認
	assert.Equal(t, 10, abils.Vitality.Total, "体力のTotal値が正しく設定されていない")
	assert.Equal(t, 8, abils.Strength.Total, "力のTotal値が正しく設定されていない")

	// HP/SPが正しく計算されたことを確認
	expectedHP := int(30 + float64(10*8+8+7)*1.0) // 30 + 95 = 125
	expectedSP := int(float64(10*2+6+9) * 1.0)    // 35
	assert.Equal(t, expectedHP, pools.HP.Max, "最大HPが正しく計算されていない")
	assert.Equal(t, expectedHP, pools.HP.Current, "現在HPが最大HPと一致していない")
	assert.Equal(t, expectedSP, pools.SP.Max, "最大SPが正しく計算されていない")
	assert.Equal(t, expectedSP, pools.SP.Current, "現在SPが最大SPと一致していない")

	// クリーンアップ
	world.Manager.DeleteEntity(entity)
}

func TestSpawnNPCHasAIMoveFSM(t *testing.T) {
	t.Parallel()
	// NPCが敵として認識されるAIMoveFSMコンポーネントを持つことを確認
	world := testutil.InitTestWorld(t)

	// SpriteSheetsを初期化
	spriteSheets := make(map[string]gc.SpriteSheet)
	spriteSheets["field"] = gc.SpriteSheet{
		Sprites: map[string]gc.Sprite{
			"void":         {Width: 32, Height: 32},
			"wall_generic": {Width: 32, Height: 32},
			"floor":        {Width: 32, Height: 32},
			"player":       {Width: 32, Height: 32},
			"warp_next":    {Width: 32, Height: 32},
			"warp_escape":  {Width: 32, Height: 32},
			"red_ball":     {Width: 32, Height: 32}, // 敵のスプライト
		},
	}
	world.Resources.SpriteSheets = &spriteSheets

	// NPCを生成（タイル座標で指定）
	_, err := SpawnEnemy(world, 5, 5, "火の玉")
	require.NoError(t, err)

	// AIMoveFSMコンポーネントを持つエンティティが存在することを確認
	enemyFound := false
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.AIMoveFSM,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		enemyFound = true
	}))

	assert.True(t, enemyFound, "SpawnEnemyで生成されたエンティティはAIMoveFSMコンポーネントを持つべき")
}

func TestSpawnEnemy_WithBoss(t *testing.T) {
	t.Parallel()

	initSpriteSheets := func(world w.World) {
		spriteSheets := make(map[string]gc.SpriteSheet)
		spriteSheets["field"] = gc.SpriteSheet{
			Sprites: map[string]gc.Sprite{
				"red_ball": {Width: 32, Height: 32},
			},
		}
		world.Resources.SpriteSheets = &spriteSheets
	}

	t.Run("WithBossオプションでBossコンポーネントが付与される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		initSpriteSheets(world)
		enemy, err := SpawnEnemy(world, 5, 5, "火の玉", WithBoss())
		require.NoError(t, err)
		assert.True(t, enemy.HasComponent(world.Components.Boss), "Bossコンポーネントを持つべき")
	})

	t.Run("オプションなしではBossコンポーネントが付与されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		initSpriteSheets(world)
		enemy, err := SpawnEnemy(world, 6, 6, "火の玉")
		require.NoError(t, err)
		assert.False(t, enemy.HasComponent(world.Components.Boss), "Bossコンポーネントを持つべきではない")
	})
}

func TestSpawnEnemy_WithDropTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// SpriteSheetsを初期化
	spriteSheets := make(map[string]gc.SpriteSheet)
	spriteSheets["field"] = gc.SpriteSheet{
		Sprites: map[string]gc.Sprite{
			"red_ball": {Width: 32, Height: 32},
		},
	}
	world.Resources.SpriteSheets = &spriteSheets

	// 「火の玉」を生成（DropTableが定義されている敵）
	enemy, err := SpawnEnemy(world, 10, 10, "火の玉")
	require.NoError(t, err, "火の玉の生成に失敗")

	// DropTableコンポーネントが付与されていることを確認
	assert.True(t, enemy.HasComponent(world.Components.DropTable), "火の玉はDropTableコンポーネントを持つべき")

	dropTable := world.Components.DropTable.Get(enemy).(*gc.DropTable)
	assert.Equal(t, "火の玉", dropTable.Name, "DropTableの名前が正しくない")
}

func TestSpawnItem(t *testing.T) {
	t.Parallel()

	t.Run("Stackableなアイテムに複数個指定できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnItem(world, "回復薬", 5, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 5, itemComp.Count)
	})

	t.Run("Stackableでないアイテムにcount=1を指定できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 1, itemComp.Count)
	})

	t.Run("Stackableでないアイテムにcount>1を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "木刀", 2, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not stackable")
		assert.Contains(t, err.Error(), "count must be 1")
	})

	t.Run("count=0を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "木刀", 0, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count must be positive")
	})

	t.Run("負のcountを指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "木刀", -1, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count must be positive")
	})

	t.Run("存在しないアイテム名を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "存在しないアイテム", 1, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})
}

func TestSpawnDoor(t *testing.T) {
	t.Parallel()

	t.Run("縦向き扉のスポーン", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, 10, 10, gc.DoorOrientationVertical)
		require.NoError(t, err, "SpawnDoor should not return an error")

		// SpriteRenderを確認（entity=0は有効なエンティティIDなので、コンポーネントの存在でチェック）
		require.True(t, door.HasComponent(world.Components.SpriteRender))
		sprite := world.Components.SpriteRender.Get(door).(*gc.SpriteRender)
		assert.Equal(t, "field", sprite.SpriteSheetName)
		assert.Equal(t, "door_vertical_closed", sprite.SpriteKey)
		assert.Equal(t, gc.DepthNumTaller, sprite.Depth)

		// Doorコンポーネントを確認
		require.True(t, door.HasComponent(world.Components.Door))
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.False(t, doorComp.IsOpen)
		assert.Equal(t, gc.DoorOrientationVertical, doorComp.Orientation)

		// BlockPass/BlockViewを確認
		assert.True(t, door.HasComponent(world.Components.BlockPass))
		assert.True(t, door.HasComponent(world.Components.BlockView))
	})

	t.Run("横向き扉のスポーン", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, 10, 10, gc.DoorOrientationHorizontal)
		require.NoError(t, err)

		// SpriteRenderを確認
		sprite := world.Components.SpriteRender.Get(door).(*gc.SpriteRender)
		assert.Equal(t, "door_horizontal_closed", sprite.SpriteKey)

		// Doorコンポーネントを確認
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.Equal(t, gc.DoorOrientationHorizontal, doorComp.Orientation)
	})
}

func TestLockAllDoors(t *testing.T) {
	t.Parallel()

	t.Run("全扉を閉じてロックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door1, err := SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		door2, err := SpawnDoor(world, 6, 6, gc.DoorOrientationVertical)
		require.NoError(t, err)

		locked := LockAllDoors(world)

		assert.Equal(t, 2, locked)
		assert.True(t, world.Components.Door.Get(door1).(*gc.Door).Locked)
		assert.True(t, world.Components.Door.Get(door2).(*gc.Door).Locked)
	})

	t.Run("開いた扉を閉じてからロックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		require.NoError(t, OpenDoor(world, door))

		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.True(t, doorComp.IsOpen)

		locked := LockAllDoors(world)

		assert.Equal(t, 1, locked)
		assert.False(t, doorComp.IsOpen, "扉が閉じられるべき")
		assert.True(t, doorComp.Locked, "扉がロックされるべき")
	})

	t.Run("既にロック済みの扉はスキップする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		world.Components.Door.Get(door).(*gc.Door).Locked = true

		locked := LockAllDoors(world)

		assert.Equal(t, 0, locked)
	})

	t.Run("扉がない場合は0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		locked := LockAllDoors(world)

		assert.Equal(t, 0, locked)
	})
}

func TestUnlockAllDoors(t *testing.T) {
	t.Parallel()

	t.Run("全扉をアンロックして開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door1, err := SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		door2, err := SpawnDoor(world, 6, 6, gc.DoorOrientationVertical)
		require.NoError(t, err)

		// ロックする
		world.Components.Door.Get(door1).(*gc.Door).Locked = true
		world.Components.Door.Get(door2).(*gc.Door).Locked = true

		opened := UnlockAllDoors(world)

		assert.Equal(t, 2, opened)
		doorComp1 := world.Components.Door.Get(door1).(*gc.Door)
		doorComp2 := world.Components.Door.Get(door2).(*gc.Door)
		assert.False(t, doorComp1.Locked)
		assert.True(t, doorComp1.IsOpen)
		assert.False(t, doorComp2.Locked)
		assert.True(t, doorComp2.IsOpen)
	})

	t.Run("既に開いている扉はカウントしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		require.NoError(t, OpenDoor(world, door))
		world.Components.Door.Get(door).(*gc.Door).Locked = true

		opened := UnlockAllDoors(world)

		assert.Equal(t, 0, opened)
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.False(t, doorComp.Locked, "アンロックされるべき")
		assert.True(t, doorComp.IsOpen, "開いたままであるべき")
	})
}

func TestDeleteDoorLockTriggers(t *testing.T) {
	t.Parallel()

	t.Run("DoorLockInteractionを持つエンティティだけ削除する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// DoorLockTriggerを2つ作成
		trigger1 := world.Manager.NewEntity()
		trigger1.AddComponent(world.Components.Interactable, &gc.Interactable{Data: gc.DoorLockInteraction{}})
		trigger2 := world.Manager.NewEntity()
		trigger2.AddComponent(world.Components.Interactable, &gc.Interactable{Data: gc.DoorLockInteraction{}})

		// 他のInteractableも作成
		other := world.Manager.NewEntity()
		other.AddComponent(world.Components.Interactable, &gc.Interactable{Data: gc.DoorInteraction{}})

		DeleteDoorLockTriggers(world)

		// DoorLockTriggerは削除されている
		count := 0
		world.Manager.Join(world.Components.Interactable).Visit(ecs.Visit(func(entity ecs.Entity) {
			interactable := world.Components.Interactable.Get(entity).(*gc.Interactable)
			if _, ok := interactable.Data.(gc.DoorLockInteraction); ok {
				count++
			}
		}))
		assert.Equal(t, 0, count, "DoorLockTriggerは全削除されるべき")

		// 他のInteractableは残っている
		assert.True(t, other.HasComponent(world.Components.Interactable), "DoorInteractionは残るべき")
	})

	t.Run("対象がない場合でもエラーにならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		assert.NotPanics(t, func() {
			DeleteDoorLockTriggers(world)
		})
	})
}

func TestMovePlayerToPosition(t *testing.T) {
	t.Parallel()

	t.Run("正常にプレイヤーの位置を更新できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		player.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
		player.AddComponent(world.Components.Camera, &gc.Camera{})

		// プレイヤーを移動
		err := MovePlayerToPosition(world, 10, 15)
		require.NoError(t, err)

		// 位置が更新されていることを確認
		gridElement := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, consts.Tile(10), gridElement.X)
		assert.Equal(t, consts.Tile(15), gridElement.Y)
	})

	t.Run("プレイヤーが存在しない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしで実行
		err := MovePlayerToPosition(world, 10, 15)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	})

	t.Run("必須コンポーネントが欠けている場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
		player.AddComponent(world.Components.Camera, &gc.Camera{})

		err := MovePlayerToPosition(world, 10, 15)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	})
}

func TestCalculateSpeed(t *testing.T) {
	t.Parallel()

	t.Run("基本Speed（能力値なし）", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()

		speed := CalculateSpeed(world, entity)
		// 基本値100、能力値なし
		assert.Equal(t, 100, speed)
	})

	t.Run("能力値によるボーナス", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Abilities, &gc.Abilities{
			Agility:   gc.Ability{Total: 10},
			Dexterity: gc.Ability{Total: 5},
		})

		speed := CalculateSpeed(world, entity)
		// 基本100 + AGI*2 (20) + DEX*1 (5) = 125
		assert.Equal(t, 125, speed)
	})

	t.Run("空腹によるペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Hunger, &gc.Hunger{Pool: gc.Pool{Current: 20, Max: 100}}) // 飢餓状態

		speed := CalculateSpeed(world, entity)
		// 基本100 - 飢餓ペナルティ50 = 50
		assert.Equal(t, 50, speed)
	})

	t.Run("過積載によるペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Pools, &gc.Pools{
			Weight: gc.PoolFloat{Max: 100, Current: 150}, // 50%超過
		})

		speed := CalculateSpeed(world, entity)
		// 基本100 - 超過ペナルティ(50*25/100=12) = 88
		assert.Equal(t, 88, speed)
	})

	t.Run("体温異常によるペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity, err := SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 通常時のSpeedを記録
		normalSpeed := CalculateSpeed(world, entity)

		// 低体温を設定してCharModifiersを再計算
		hs := world.Components.HealthStatus.Get(entity).(*gc.HealthStatus)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityMedium,
		})
		skills := world.Components.Skills.Get(entity).(*gc.Skills)
		abils := world.Components.Abilities.Get(entity).(*gc.Abilities)
		mods := gc.RecalculateCharModifiers(skills, abils, hs)
		entity.AddComponent(world.Components.CharModifiers, mods)

		coldSpeed := CalculateSpeed(world, entity)
		t.Logf("normalSpeed=%d coldSpeed=%d hasMods=%v moveCost=%d", normalSpeed, coldSpeed, entity.HasComponent(world.Components.CharModifiers), mods.MoveCost)
		assert.Less(t, coldSpeed, normalSpeed, "低体温によりSpeedが低下するべき")
	})

	t.Run("複合ペナルティで最小値に達する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Hunger, &gc.Hunger{Pool: gc.Pool{Current: 5, Max: 100}}) // 餓死寸前(-75)
		entity.AddComponent(world.Components.Pools, &gc.Pools{
			Weight: gc.PoolFloat{Max: 100, Current: 400}, // 大幅超過（最大-75）
		})

		speed := CalculateSpeed(world, entity)
		// ペナルティが大きくても最小値25を下回らない
		assert.Equal(t, 25, speed)
	})
}

func TestHungerSpeedPenalty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hunger   int
		expected int
	}{
		{"満腹", 100, 0},
		{"やや空腹", 60, -10},
		{"空腹", 30, -25},
		{"飢餓", 15, -50},
		{"餓死寸前", 5, -75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			penalty := hungerSpeedPenalty(tt.hunger)
			assert.Equal(t, tt.expected, penalty)
		})
	}
}

func TestOverweightPenalty(t *testing.T) {
	t.Parallel()

	t.Run("超過なし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Pools, &gc.Pools{
			Weight: gc.PoolFloat{Max: 100, Current: 80},
		})

		penalty := calculateOverweightPenalty(world, entity)
		assert.Equal(t, 0, penalty)
	})

	t.Run("50%超過", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Pools, &gc.Pools{
			Weight: gc.PoolFloat{Max: 100, Current: 150},
		})

		penalty := calculateOverweightPenalty(world, entity)
		// 50 * 25 / 100 = 12.5 -> -12
		assert.Equal(t, -12, penalty)
	})

	t.Run("最大ペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Pools, &gc.Pools{
			Weight: gc.PoolFloat{Max: 100, Current: 500}, // 400%超過
		})

		penalty := calculateOverweightPenalty(world, entity)
		// 最大-75
		assert.Equal(t, -75, penalty)
	})
}
