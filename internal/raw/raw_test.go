package raw

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeRaws(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "リペア"
Description = "半分程度回復する"

[[Items]]
Name = "回復薬"
Description = "半分程度回復する"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	items := PtrSlice(raws.Items)
	assert.Len(t, items, 2)
	assert.Equal(t, "リペア", items[0].Name)
	assert.Equal(t, "回復薬", items[1].Name)
}

func TestGenerateItem(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "リペア"
SpriteSheetName = "field"
SpriteKey = "repair_item"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)
	entitySpec, err := NewItemSpec(raws, "リペア")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.Name)
	assert.NotNil(t, entitySpec.Description)
	assert.NotNil(t, entitySpec.SpriteRender)
}

func TestGenerateItemWithoutSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "テストアイテム"
Description = "スプライトなしアイテム"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// 現在の実装ではスプライト情報なしでも生成される（デフォルト値が設定される）
	entitySpec, err := NewItemSpec(raws, "テストアイテム")
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName)
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey)
}

func TestGenerateMemberWithSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "テストプレイヤー"
Player = true
SpriteSheetName = "field"
SpriteKey = "player"
[Members.Abilities]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)
	entitySpec, err := NewPlayerSpec(raws, "テストプレイヤー")
	require.NoError(t, err)

	// 基本コンポーネントの確認
	assert.NotNil(t, entitySpec.Name)
	assert.NotNil(t, entitySpec.Player)

	// SpriteRenderコンポーネントの確認
	assert.NotNil(t, entitySpec.SpriteRender, "SpriteRenderコンポーネントが設定されていない")
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName, "SpriteSheetNameが正しくない")
	assert.Equal(t, "player", entitySpec.SpriteRender.SpriteKey, "SpriteKeyが正しくない")
	assert.Equal(t, gc.DepthNumPlayer, entitySpec.SpriteRender.Depth, "Depthが正しくない")
}

func TestGenerateMemberWithoutSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "スプライトなしキャラ"
Player = true
[Members.Abilities]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// 現在の実装ではスプライト情報なしでも生成される（空文字列が設定される）
	entitySpec, err := NewPlayerSpec(raws, "スプライトなしキャラ")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Empty(t, entitySpec.SpriteRender.SpriteSheetName)
	assert.Empty(t, entitySpec.SpriteRender.SpriteKey)
}

func TestGenerateMaterialWithSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "テスト素材"
Description = "スプライト付き素材"
SpriteSheetName = "field"
SpriteKey = "field_item"
Stackable = true
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)
	entitySpec, err := NewItemSpec(raws, "テスト素材")
	require.NoError(t, err)

	// 基本コンポーネントの確認
	assert.NotNil(t, entitySpec.Name)
	assert.NotNil(t, entitySpec.Description)
	// Stackable=trueのアイテムにはStackableコンポーネントが付与される
	assert.NotNil(t, entitySpec.Stackable)

	// SpriteRenderコンポーネントの確認
	assert.NotNil(t, entitySpec.SpriteRender, "SpriteRenderコンポーネントが設定されていない")
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName, "SpriteSheetNameが正しくない")
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey, "SpriteKeyが正しくない")
	assert.Equal(t, gc.DepthNumRug, entitySpec.SpriteRender.Depth, "Depthが正しくない")
}

func TestGenerateMaterialWithoutSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "スプライトなし素材"
Description = "スプライトなし素材"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// 現在の実装ではスプライト情報なしでも生成される（デフォルト値が設定される）
	entitySpec, err := NewItemSpec(raws, "スプライトなし素材")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName)
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey)
	// Stackable=true が設定されていないので Stackable コンポーネントは付かない
	assert.Nil(t, entitySpec.Stackable)
}

func TestLoadTilesFromRaw(t *testing.T) {
	t.Parallel()

	// テスト用のTOMLデータ（タイル定義を含む）
	tomlData := `
[[Tiles]]
Name = "TestFloor"
Description = "テスト用床タイル"

[[Tiles]]
Name = "TestWall"
Description = "テスト用壁タイル"
BlockPass = true

[[Items]]
Name = "テストアイテム"
Description = "テスト用"
`

	raws, err := DecodeRaws(tomlData)
	require.NoError(t, err, "raw.goからの読み込みに失敗")

	// 基本的なタイルが定義されていることを確認
	expectedTiles := []string{"TestFloor", "TestWall"}
	for _, tileName := range expectedTiles {
		_, err := GetTile(raws, tileName)
		assert.NoError(t, err, "タイル '%s' が見つかりません", tileName)
	}
}

func TestGenerateTileFromRaw(t *testing.T) {
	t.Parallel()

	tomlData := `
[[Tiles]]
Name = "GenerateTestFloor"
Description = "生成テスト用床タイル"

[[Tiles]]
Name = "GenerateTestWall"
Description = "生成テスト用壁タイル"
BlockPass = true
`

	raws, err := DecodeRaws(tomlData)
	require.NoError(t, err, "テストTOMLの読み込みに失敗")

	// 床タイルの取得をテスト
	floorTile, err := GetTile(raws, "GenerateTestFloor")
	require.NoError(t, err, "床タイルの取得に失敗")
	assert.False(t, floorTile.BlockPass)

	// 壁タイルの取得をテスト
	wallTile, err := GetTile(raws, "GenerateTestWall")
	require.NoError(t, err, "壁タイルの取得に失敗")
	assert.True(t, wallTile.BlockPass)

	// 存在しないタイルのテスト（エラーが発生する）
	_, err = GetTile(raws, "NonExistent")
	assert.Error(t, err, "存在しないタイルでエラーが発生すべき")
}

// TestGenerateTileSpecFromRaw - TileSpecは削除されたためこのテストは不要

func TestTileHelperFunctionsFromRaw(t *testing.T) {
	t.Parallel()

	tomlData := `
[[Tiles]]
Name = "Helper1"
Description = "ヘルパー関数テスト1"

[[Tiles]]
Name = "Helper2"
Description = "ヘルパー関数テスト2"
BlockPass = true
`

	raws, err := DecodeRaws(tomlData)
	require.NoError(t, err, "テストTOMLの読み込みに失敗")

	// GetTile のテスト（存在するタイル）
	tileRaw, err := GetTile(raws, "Helper1")
	require.NoError(t, err, "タイル取得に失敗")
	assert.False(t, tileRaw.BlockPass)

	// GetTile のテスト（存在しないタイル）
	_, err = GetTile(raws, "NonExistent")
	assert.Error(t, err, "存在しないタイルでエラーが発生すべき")
}

func TestTilePropertiesFromRaw(t *testing.T) {
	t.Parallel()

	// Walkableフィールドのテスト
	tomlData := `
[[Tiles]]
Name = "EmptyLike"
Description = "空のような性質"
BlockPass = true

[[Tiles]]
Name = "FloorLike"
Description = "床のような性質"

[[Tiles]]
Name = "WallLike"
Description = "壁のような性質"
BlockPass = true
`

	raws, err := DecodeRaws(tomlData)
	require.NoError(t, err, "テストTOMLの読み込みに失敗")

	testCases := []struct {
		name         string
		expectedWalk bool
	}{
		{"EmptyLike", false},
		{"FloorLike", true},
		{"WallLike", false},
	}

	for _, tc := range testCases {
		tile, err := GetTile(raws, tc.name)
		require.NoError(t, err, "タイル取得に失敗: %s", tc.name)
		actualWalk := !tile.BlockPass
		assert.Equal(t, tc.expectedWalk, actualWalk, "Walkableが期待値と一致しない: %s", tc.name)
	}
}

func TestLoadFromRealTileFile(t *testing.T) {
	t.Parallel()

	// 実際のraw.tomlファイルからタイル定義を読み込み
	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "実際のraw.tomlファイルの読み込みに失敗")

	// 基本的なタイルが定義されていることを確認
	expectedTiles := []string{"floor", "wall", "dirt"}
	for _, tileName := range expectedTiles {
		_, err := GetTile(raws, tileName)
		require.NoError(t, err, "タイル '%s' が実際のファイルで見つかりません", tileName)
	}

	// 実際のタイル取得テスト
	floorTile, err := GetTile(raws, "floor")
	require.NoError(t, err, "床タイル取得に失敗")
	assert.False(t, floorTile.BlockPass)

	// 壁タイルテスト
	wallTile, err := GetTile(raws, "wall")
	require.NoError(t, err, "壁タイル取得に失敗")
	assert.True(t, wallTile.BlockPass)
}

func TestLoadWithUnknownFields(t *testing.T) {
	t.Parallel()

	// 未知のフィールドを含むTOMLデータ
	invalidToml := `
[[Items]]
Name = "テストアイテム"
Description = "正常なアイテム"
UnknownField = "これは未知のフィールド"

[[UnknownSection]]
SomeField = "これは未知のセクション"
`

	_, err := DecodeRaws(invalidToml)
	require.Error(t, err, "未知のフィールドがあるTOMLでエラーが発生すべき")
	assert.Contains(t, err.Error(), "unknown keys found in TOML", "エラーメッセージに未知のキーについての情報が含まれるべき")
}

func TestLoadWithValidFields(t *testing.T) {
	t.Parallel()

	// 正常なTOMLデータ（既知のフィールドのみ）
	validToml := `
[[Items]]
Name = "テストアイテム"
Description = "正常なアイテム"
SpriteSheetName = "test_sheet"
SpriteKey = "test_key"

[[Tiles]]
Name = "テストタイル"
Description = "正常なタイル"
`

	raws, err := DecodeRaws(validToml)
	require.NoError(t, err, "正常なTOMLでエラーが発生してはいけない")
	assert.Len(t, PtrSlice(raws.Items), 1, "アイテムが1つ読み込まれるべき")
	assert.Len(t, PtrSlice(raws.Tiles), 1, "タイルが1つ読み込まれるべき")
}

func TestItemWithAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "アニメーションアイテム"
Description = "2フレームアニメーションするアイテム"
SpriteSheetName = "field"
SpriteKey = "item_0"
AnimKeys = ["item_0", "item_1"]
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// AnimKeysが正しく読み込まれていることを確認
	items := PtrSlice(raws.Items)
	assert.Len(t, items, 1)
	item := items[0]
	assert.Equal(t, []string{"item_0", "item_1"}, PtrSlice(item.AnimKeys))

	// NewItemSpecでAnimKeysがSpriteRenderに設定されることを確認
	entitySpec, err := NewItemSpec(raws, "アニメーションアイテム")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, []string{"item_0", "item_1"}, entitySpec.SpriteRender.AnimKeys)
}

func TestItemWithoutAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "静的アイテム"
Description = "アニメーションしないアイテム"
SpriteSheetName = "field"
SpriteKey = "static_item"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// AnimKeysが指定されていない場合はnil
	item := PtrSlice(raws.Items)[0]
	assert.Nil(t, item.AnimKeys)

	// NewItemSpecでもAnimKeysはnil
	entitySpec, err := NewItemSpec(raws, "静的アイテム")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Nil(t, entitySpec.SpriteRender.AnimKeys)
}

func TestMemberWithAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "アニメーションキャラ"
Player = true
SpriteSheetName = "field"
SpriteKey = "player_0"
AnimKeys = ["player_0", "player_1"]
[Members.Abilities]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// AnimKeysが正しく読み込まれていることを確認
	members := PtrSlice(raws.Members)
	assert.Len(t, members, 1)
	member := members[0]
	assert.Equal(t, []string{"player_0", "player_1"}, PtrSlice(member.AnimKeys))

	// NewPlayerSpecでAnimKeysがSpriteRenderに設定されることを確認
	entitySpec, err := NewPlayerSpec(raws, "アニメーションキャラ")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, []string{"player_0", "player_1"}, entitySpec.SpriteRender.AnimKeys)
}

func TestMemberWithoutAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "静的キャラ"
Player = true
SpriteSheetName = "field"
SpriteKey = "static_player"
[Members.Abilities]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// AnimKeysが指定されていない場合はnil
	member := PtrSlice(raws.Members)[0]
	assert.Nil(t, member.AnimKeys)

	// NewPlayerSpecでもAnimKeysはnil
	entitySpec, err := NewPlayerSpec(raws, "静的キャラ")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Nil(t, entitySpec.SpriteRender.AnimKeys)
}

func TestPropWithAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "アニメーションProp"
Description = "2フレームアニメーションする置物"
BlockPass = false
BlockView = false
AnimKeys = ["fire_0_", "fire_1_"]

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "fire_0_"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// AnimKeysが正しく読み込まれていることを確認
	props := PtrSlice(raws.Props)
	assert.Len(t, props, 1)
	prop := props[0]
	assert.Equal(t, []string{"fire_0_", "fire_1_"}, PtrSlice(prop.AnimKeys))

	// NewPropSpecでAnimKeysがSpriteRenderに設定されることを確認
	entitySpec, err := NewPropSpec(raws, "アニメーションProp")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, []string{"fire_0_", "fire_1_"}, entitySpec.SpriteRender.AnimKeys)
}

func TestPropWithoutAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "静的Prop"
Description = "アニメーションしない置物"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// AnimKeysが指定されていない場合はnil
	prop := PtrSlice(raws.Props)[0]
	assert.Nil(t, prop.AnimKeys)

	// NewPropSpecでもAnimKeysはnil
	entitySpec, err := NewPropSpec(raws, "静的Prop")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Nil(t, entitySpec.SpriteRender.AnimKeys)
}

func TestPropBlockPassWithPassCostIsError(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "矛盾Prop"
Description = "通行不可なのに移動コストがある"
BlockPass = true
BlockView = false
PassCost = 100

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewPropSpec(raws, "矛盾Prop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blockPassとpassCostは同時に設定できません")
}

func TestPropWithHP(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "壊れるProp"
Description = "破壊可能な置物"
BlockPass = true
BlockView = false
Hp = 20

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "壊れるProp")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.HP)
	assert.Equal(t, 20, entitySpec.HP.Max)
	assert.Equal(t, 20, entitySpec.HP.Current)
	assert.NotNil(t, entitySpec.Interactable)
	assert.NotEmpty(t, entitySpec.Interactable.Interactions, "HPを持つPropにはInteractionsが設定されるべき")
	ok := entitySpec.Interactable.Interactions[0] == gc.InteractionMelee
	assert.True(t, ok, "HPを持つPropにはMeleeInteractionが設定されるべき")
}

func TestPropWithoutHP(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "壊れないProp"
Description = "破壊不能な置物"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "壊れないProp")
	require.NoError(t, err)
	assert.Nil(t, entitySpec.WeightCapacity)
	assert.Nil(t, entitySpec.Interactable, "HPを持たないPropにはInteractableが設定されないべき")
}

func TestPropWithStorage(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "木箱"
Description = "古びた木箱"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "wooden_chest"
Depth = 2

[Props.Storage]
MaxWeight = 20.0
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "木箱")
	require.NoError(t, err)

	assert.NotNil(t, entitySpec.WeightCapacity, "Storage付きPropにはWeightCapacityコンポーネントが設定されるべき")
	assert.Equal(t, 20.0, entitySpec.WeightCapacity.Max)

	require.NotNil(t, entitySpec.Interactable, "Storage付きPropにはInteractableが設定されるべき")
	assert.NotEmpty(t, entitySpec.Interactable.Interactions, "Storage付きPropにはInteractionsが設定されるべき")
	ok := entitySpec.Interactable.Interactions[0] == gc.InteractionStorage
	assert.True(t, ok, "Storage付きPropにはStorageInteractionが設定されるべき")
}

func TestPropWithoutStorage(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "テーブル"
Description = "普通のテーブル"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "table"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "テーブル")
	require.NoError(t, err)

	assert.Nil(t, entitySpec.WeightCapacity, "Storage定義のないPropにはWeightCapacityコンポーネントが設定されないべき")
}

func TestMemberCombatPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		combatPolicy    string
		expectedDefault gc.CombatPolicy
		expectedCurrent gc.CombatPolicy
	}{
		{"attack", string(gc.CombatAttack), gc.CombatAttack, gc.CombatAttack},
		{"ignore", string(gc.CombatIgnore), gc.CombatIgnore, gc.CombatIgnore},
		{"evade", string(gc.CombatEvade), gc.CombatEvade, gc.CombatEvade},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			toml := `
[[Members]]
Name = "テスト敵"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0", "enemy_1"]
CommandTableName = ""
DropTableName = ""
CombatPolicy = "` + tt.combatPolicy + `"
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2
`
			raws, err := DecodeRaws(toml)
			require.NoError(t, err)

			entitySpec, err := NewMemberSpec(raws, "テスト敵")
			require.NoError(t, err)
			require.NotNil(t, entitySpec.SoloAI)
			solo := entitySpec.SoloAI
			assert.Equal(t, tt.expectedDefault, solo.CombatDefault)
			assert.Equal(t, tt.expectedCurrent, solo.CombatCurrent)
		})
	}
}

func TestMemberCombatPolicyUnset(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "態度なし"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0"]
CommandTableName = ""
DropTableName = ""
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewMemberSpec(raws, "態度なし")
	require.NoError(t, err)
	require.NotNil(t, entitySpec.SoloAI)
	assert.Equal(t, gc.CombatAttack, entitySpec.SoloAI.CombatDefault)
}

func TestMemberMovementPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strategy string
		expected gc.SoloMovement
	}{
		{"random", "random", gc.SoloRandom},
		{"stationary", "stationary", gc.SoloStationary},
		{"wander", "wander", gc.SoloWander},
		{"patrol", "patrol", gc.SoloPatrol},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			toml := `
[[Members]]
Name = "テスト敵"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0"]
CommandTableName = ""
DropTableName = ""
movementPattern = "` + tt.strategy + `"
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2
`
			raws, err := DecodeRaws(toml)
			require.NoError(t, err)

			entitySpec, err := NewMemberSpec(raws, "テスト敵")
			require.NoError(t, err)
			require.NotNil(t, entitySpec.SoloAI)
			assert.Equal(t, tt.expected, entitySpec.SoloAI.Movement)
		})
	}
}

func TestNewWeaponSpec(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "テスト剣"
Description = "テスト用の武器"
SpriteSheetName = "field"
SpriteKey = "sword"

[Items.Melee]
Damage = 10
Accuracy = 90
AttackCount = 1
Element = "none"
AttackCategory = "SWORD"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewWeaponSpec(raws, "テスト剣")
	require.NoError(t, err)
	assert.NotNil(t, spec.Melee)
	assert.Equal(t, 10, spec.Melee.Damage)
}

func TestNewWeaponSpec_NotAWeapon(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "回復薬"
Description = "回復するアイテム"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewWeaponSpec(raws, "回復薬")
	assert.Error(t, err, "Melee/Fireを持たないアイテムは武器ではない")
}

func TestNewWeaponSpec_NotFound(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "何か"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewWeaponSpec(raws, "存在しない")
	assert.Error(t, err)
}

func TestNewEnemySpec(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "テスト敵"
SpriteSheetName = "field"
SpriteKey = "enemy"
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewEnemySpec(raws, "テスト敵")
	require.NoError(t, err)
	assert.NotNil(t, spec.FactionEnemy, "敵は敵派閥に属する")
	assert.Nil(t, spec.Player, "敵はPlayerではない")
}

func TestNewTileSpec(t *testing.T) {
	t.Parallel()

	str := `
[[Tiles]]
Name = "test_floor"
Description = "テスト床"
BlockPass = false
BlockView = false
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewTileSpec(raws, "test_floor", 5, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, "test_floor", spec.Name.Name)
	assert.NotNil(t, spec.GridElement)
	assert.Nil(t, spec.BlockPass, "通行可能なタイルにはBlockPassがない")
	assert.Nil(t, spec.BlockView, "視界を遮らないタイルにはBlockViewがない")
}

func TestNewTileSpec_WithBlockPass(t *testing.T) {
	t.Parallel()

	str := `
[[Tiles]]
Name = "test_wall"
Description = "テスト壁"
BlockPass = true
BlockView = true
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewTileSpec(raws, "test_wall", 3, 7, nil)
	require.NoError(t, err)
	assert.NotNil(t, spec.BlockPass, "壁タイルにはBlockPassがある")
	assert.NotNil(t, spec.BlockView, "壁タイルにはBlockViewがある")
}

func TestNewTileSpec_WithAutoTileIndex(t *testing.T) {
	t.Parallel()

	str := `
[[Tiles]]
Name = "auto_tile"
Description = "オートタイル"

[Tiles.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "wall"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	idx := 5
	spec, err := NewTileSpec(raws, "auto_tile", 0, 0, &idx)
	require.NoError(t, err)
	assert.Equal(t, "wall_5", spec.SpriteRender.SpriteKey)
}

func TestGetItemTable(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	// ItemTablesが存在する場合のテスト
	if len(PtrSlice(raws.ItemTables)) > 0 {
		tableName := PtrSlice(raws.ItemTables)[0].Name
		table, err := GetItemTable(raws, tableName)
		require.NoError(t, err)
		assert.Equal(t, tableName, table.Name)
	}

	// 存在しないテーブル
	_, err = GetItemTable(raws, "存在しないテーブル")
	assert.Error(t, err)
}

func TestGetEnemyTable(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	// EnemyTablesが存在する場合のテスト
	if len(PtrSlice(raws.EnemyTables)) > 0 {
		tableName := PtrSlice(raws.EnemyTables)[0].Name
		table, err := GetEnemyTable(raws, tableName)
		require.NoError(t, err)
		assert.Equal(t, tableName, table.Name)
	}

	// 存在しないテーブル
	_, err = GetEnemyTable(raws, "存在しないテーブル")
	assert.Error(t, err)
}

func TestMemberMovementPatternUnset(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "パターンなし"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0"]
CommandTableName = ""
DropTableName = ""
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewMemberSpec(raws, "パターンなし")
	require.NoError(t, err)
	require.NotNil(t, entitySpec.SoloAI)
	assert.Empty(t, entitySpec.SoloAI.Movement)
}
