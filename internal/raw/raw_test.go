package raw

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/kijimaD/ruins/assets"
	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/BurntSushi/toml"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeRaws(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "リペア"
id = "リペア"
Description = "半分程度回復する"

[[Items]]
Name = "回復薬"
id = "回復薬"
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
id = "リペア"
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
id = "テストアイテム"
Description = "スプライトなしアイテム"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// 現在の実装ではスプライト情報なしでも生成される（デフォルト値が設定される）
	entitySpec, err := NewItemSpec(raws, "テストアイテム")
	require.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName)
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey)
}

func TestGenerateMemberWithSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "テストプレイヤー"
id = "テストプレイヤー"
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
id = "スプライトなしキャラ"
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
id = "テスト素材"
Description = "スプライト付き素材"
SpriteSheetName = "field"
SpriteKey = "field_item"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)
	entitySpec, err := NewItemSpec(raws, "テスト素材")
	require.NoError(t, err)

	// 基本コンポーネントの確認
	assert.NotNil(t, entitySpec.Name)
	assert.NotNil(t, entitySpec.Description)

	// SpriteRenderコンポーネントの確認
	assert.NotNil(t, entitySpec.SpriteRender, "SpriteRenderコンポーネントが設定されていない")
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName, "SpriteSheetNameが正しくない")
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey, "SpriteKeyが正しくない")
	assert.Equal(t, gc.DepthNumItem, entitySpec.SpriteRender.Depth, "Depthが正しくない")
}

func TestGenerateMaterialWithoutSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "スプライトなし素材"
id = "スプライトなし素材"
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
}

func TestLoadTilesFromRaw(t *testing.T) {
	t.Parallel()

	// テスト用のTOMLデータ（タイル定義を含む）
	tomlData := `
[[Tiles]]
Name = "TestFloor"
id = "TestFloor"
Description = "テスト用床タイル"

[[Tiles]]
Name = "TestWall"
id = "TestWall"
Description = "テスト用壁タイル"
BlockPass = true

[[Items]]
Name = "テストアイテム"
id = "テストアイテム"
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
id = "GenerateTestFloor"
Description = "生成テスト用床タイル"

[[Tiles]]
Name = "GenerateTestWall"
id = "GenerateTestWall"
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
id = "Helper1"
Description = "ヘルパー関数テスト1"

[[Tiles]]
Name = "Helper2"
id = "Helper2"
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
id = "EmptyLike"
Description = "空のような性質"
BlockPass = true

[[Tiles]]
Name = "FloorLike"
id = "FloorLike"
Description = "床のような性質"

[[Tiles]]
Name = "WallLike"
id = "WallLike"
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
id = "テストアイテム"
Description = "正常なアイテム"
UnknownField = "これは未知のフィールド"

[[UnknownSection]]
SomeField = "これは未知のセクション"
`

	_, err := DecodeRaws(invalidToml)
	require.ErrorIs(t, err, errUnknownTOMLKeys)
}

func TestLoadWithValidFields(t *testing.T) {
	t.Parallel()

	// 正常なTOMLデータ（既知のフィールドのみ）
	validToml := `
[[Items]]
Name = "テストアイテム"
id = "テストアイテム"
Description = "正常なアイテム"
SpriteSheetName = "test_sheet"
SpriteKey = "test_key"

[[Tiles]]
Name = "テストタイル"
id = "テストタイル"
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
id = "アニメーションアイテム"
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
id = "静的アイテム"
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
id = "アニメーションキャラ"
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
id = "静的キャラ"
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
id = "アニメーションProp"
Description = "2フレームアニメーションする置物"
BlockPass = false
BlockView = false
AnimKeys = ["fire_0_", "fire_1_"]

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "fire_0_"
Depth = "decal"
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
id = "静的Prop"
Description = "アニメーションしない置物"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = "decal"
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
id = "矛盾Prop"
Description = "通行不可なのに移動コストがある"
BlockPass = true
BlockView = false
PassCost = 100

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = "decal"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewPropSpec(raws, "矛盾Prop")
	require.ErrorIs(t, err, errBlockPassWithPassCost)
}

func TestPropWithHP(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "壊れるProp"
id = "壊れるProp"
Description = "破壊可能な置物"
BlockPass = true
BlockView = false
Hp = 20

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = "decal"
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

func TestPropFromHP(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "木の机"
id = "木の机"
Description = "殴れば壊れる机"
BlockPass = true
BlockView = false
hp = 30

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = "decal"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "木の机")
	require.NoError(t, err)
	require.NotNil(t, entitySpec.HP, "prop は hp から HP を得る")
	assert.Equal(t, 30, entitySpec.HP.Max)
	require.NotNil(t, entitySpec.Interactable)
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionMelee, "全 prop は殴って壊せる")
}

func TestPropWithStorage(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "木箱"
id = "木箱"
Description = "古びた木箱"
BlockPass = true
BlockView = false
hp = 30

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "wooden_chest"
Depth = "rug"

[Props.Storage]
MaxWeight = "20 kg"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "木箱")
	require.NoError(t, err)

	assert.NotNil(t, entitySpec.WeightCapacity, "Storage付きPropにはWeightCapacityコンポーネントが設定されるべき")
	assert.Equal(t, consts.MustParseWeight("20 kg"), entitySpec.WeightCapacity.Max)

	require.NotNil(t, entitySpec.Interactable, "Storage付きPropにはInteractableが設定されるべき")
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionStorage, "Storage付きPropにはStorageInteractionが設定されるべき")
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionMelee, "全 prop は殴って壊せる")
}

func TestPropWithWarpPrev(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "上り階段"
id = "上り階段"
Description = "1つ上の階層へ戻るポータル"
BlockPass = false
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "warp_next_0"
Depth = "decal"

[Props.WarpPrevTrigger]
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "上り階段")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.Interactable, "WarpPrevTrigger付きPropにはInteractableが設定されるべき")
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionPortalPrev,
		"WarpPrevTrigger付きPropは InteractionPortalPrev を持つべき")
}

func TestPropWithoutStorage(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "テーブル"
id = "テーブル"
Description = "普通のテーブル"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "table"
Depth = "decal"
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
		expectedDefault rawschema.CombatPolicyType
		expectedCurrent rawschema.CombatPolicyType
	}{
		{"attack", string(rawschema.Attack), rawschema.Attack, rawschema.Attack},
		{"ignore", string(rawschema.Ignore), rawschema.Ignore, rawschema.Ignore},
		{"evade", string(rawschema.Evade), rawschema.Evade, rawschema.Evade},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			toml := `
[[Members]]
Name = "テスト敵"
id = "テスト敵"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0", "enemy_1"]
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
id = "態度なし"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0"]
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
	assert.Equal(t, rawschema.Attack, entitySpec.SoloAI.CombatDefault)
}

func TestMemberMovementPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strategy string
		expected rawschema.MovementPatternType
	}{
		{"random", "random", rawschema.Random},
		{"stationary", "stationary", rawschema.Stationary},
		{"wander", "wander", rawschema.Wander},
		{"patrol", "patrol", rawschema.Patrol},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			toml := `
[[Members]]
Name = "テスト敵"
id = "テスト敵"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0"]
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
id = "テスト剣"
Description = "テスト用の武器"
SpriteSheetName = "field"
SpriteKey = "sword"

[Items.Melee]
Damage = 10
Accuracy = 90
AttackCount = 1
Element = "none"
AttackCategory = "sword"
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
id = "回復薬"
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
id = "何か"
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
id = "テスト敵"
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
id = "test_floor"
Description = "テスト床"
BlockPass = false
BlockView = false
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewTileSpec(raws, "test_floor", 5, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, i18n.Msgid("test_floor"), spec.Name.Name)
	assert.NotNil(t, spec.GridElement)
	assert.Nil(t, spec.BlockPass, "通行可能なタイルにはBlockPassがない")
	assert.Nil(t, spec.BlockView, "視界を遮らないタイルにはBlockViewがない")
}

func TestNewTileSpec_WithBlockPass(t *testing.T) {
	t.Parallel()

	str := `
[[Tiles]]
Name = "test_wall"
id = "test_wall"
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

func TestTileIsTall_実データで描画順が高さのあるタイルだけが高い(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	var tall []string
	for _, tile := range PtrSlice(raws.Tiles) {
		if TileIsTall(tile) {
			tall = append(tall, tile.Id)
		}
	}
	want := append([]string{"wall", "dwall", "void"}, wallMaterialTiles()...)
	assert.ElementsMatch(t, want, tall)
}

func TestNewTileSpec_WithAutoTileIndex(t *testing.T) {
	t.Parallel()

	str := `
[[Tiles]]
Name = "auto_tile"
id = "auto_tile"
Description = "オートタイル"

[Tiles.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "wall"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewTileSpec(raws, "auto_tile", 0, 0, new(5))
	require.NoError(t, err)
	assert.Equal(t, "wall_5", spec.SpriteRender.SpriteKey)
}

func TestGetItemTable(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	// ItemTablesが存在する場合のテスト
	if len(PtrSlice(raws.ItemTables)) > 0 {
		tableID := PtrSlice(raws.ItemTables)[0].Id
		table, err := GetItemTable(raws, tableID)
		require.NoError(t, err)
		assert.Equal(t, tableID, table.Id)
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
		tableID := PtrSlice(raws.EnemyTables)[0].Id
		table, err := GetEnemyTable(raws, tableID)
		require.NoError(t, err)
		assert.Equal(t, tableID, table.Id)
	}

	// 存在しないテーブル
	_, err = GetEnemyTable(raws, "存在しないテーブル")
	assert.Error(t, err)
}

func TestItemName_存在するIDは表示名を返す(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "回復薬"
id = "item_heal"
Description = "半分程度回復する"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	assert.Equal(t, i18n.Msgid("回復薬"), ItemName(raws, "item_heal"))
}

func TestItemName_存在しないIDはIDをそのまま返す(t *testing.T) {
	t.Parallel()

	raws, err := DecodeRaws("")
	require.NoError(t, err)

	assert.Equal(t, i18n.Msgid("存在しないID"), ItemName(raws, "存在しないID"))
}

func TestFindMember_存在しないIDはKeyNotFoundErrorになる(t *testing.T) {
	t.Parallel()

	raws, err := DecodeRaws("")
	require.NoError(t, err)

	_, err = FindMember(raws, "存在しないメンバー")
	require.Error(t, err)

	var keyErr KeyNotFoundError
	require.ErrorAs(t, err, &keyErr)
	assert.Equal(t, "存在しないメンバー", keyErr.Key)
	assert.Equal(t, "Members", keyErr.Collection)
}

func TestFindMember_存在するIDはメンバーを返す(t *testing.T) {
	t.Parallel()

	raws := rawschema.Raws{Members: &[]rawschema.Member{{Id: "主人公"}}}

	got, err := FindMember(raws, "主人公")
	require.NoError(t, err)
	assert.Equal(t, "主人公", got.Id)
}

func TestPlayerMemberID_player持ちのIDを返す(t *testing.T) {
	t.Parallel()

	isPlayer := true
	raws := rawschema.Raws{Members: &[]rawschema.Member{
		{Id: "村人"},
		{Id: "主人公", Player: &isPlayer},
	}}

	got, err := PlayerMemberID(raws)
	require.NoError(t, err)
	assert.Equal(t, "主人公", got)
}

func TestPlayerMemberID_player持ちが無ければerrPlayerMemberMissing(t *testing.T) {
	t.Parallel()

	notPlayer := false
	raws := rawschema.Raws{Members: &[]rawschema.Member{
		{Id: "村人"},
		{Id: "商人", Player: &notPlayer},
	}}

	_, err := PlayerMemberID(raws)
	require.ErrorIs(t, err, errPlayerMemberMissing)
}

func TestFindSpriteSheet_存在しないIDはKeyNotFoundErrorになる(t *testing.T) {
	t.Parallel()

	raws, err := DecodeRaws("")
	require.NoError(t, err)

	_, err = FindSpriteSheet(raws, "存在しないシート")
	require.Error(t, err)

	var keyErr KeyNotFoundError
	require.ErrorAs(t, err, &keyErr)
	assert.Equal(t, "存在しないシート", keyErr.Key)
	assert.Equal(t, "SpriteSheets", keyErr.Collection)
}

func TestMemberMovementPatternUnset(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "パターンなし"
id = "パターンなし"
SpriteSheetName = "field"
SpriteKey = "enemy"
AnimKeys = ["enemy_0"]
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

// TestNewItemSpec_材質と重さから燃料を導く は、可燃な材質のアイテムが材質のkgあたり熱量×重量kgで
// 燃料になり、不燃の材質や材質未指定は燃料を持たないことを確認する。
func TestNewItemSpec_材質と重さから燃料を導く(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
name = "wood log"
id = "wood_log"
weight = "3 kg"
material = "WOOD"

[[Items]]
name = "iron bar"
id = "iron_bar"
weight = "2 kg"
material = "METAL"

[[Items]]
name = "no material"
id = "no_mat"
weight = "1 kg"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	// 熱量は保持せず材質と重量から導くので、ここでは材質の付与だけを確かめる。
	// 熱量の算出は query.HeatOf のテストで担う
	wood, err := NewItemSpec(raws, "wood_log")
	require.NoError(t, err)
	require.NotNil(t, wood.Material, "材質を明示したアイテムは Material を持つ")
	assert.Equal(t, rawschema.WOOD, wood.Material.Kind)

	iron, err := NewItemSpec(raws, "iron_bar")
	require.NoError(t, err)
	require.NotNil(t, iron.Material, "不燃の材質も Material は付く")
	assert.Equal(t, rawschema.METAL, iron.Material.Kind)

	none, err := NewItemSpec(raws, "no_mat")
	require.NoError(t, err)
	assert.Nil(t, none.Material, "材質未指定は Material を持たない")
}

func TestDecodeRaws_TOML構文エラーはエラーを返す(t *testing.T) {
	t.Parallel()

	// 配列テーブルの閉じ括弧が1つ欠けており、TOMLの構文として壊れている
	brokenToml := `
[[Items]
Name = "壊れたアイテム"
id = "壊れたアイテム"
`
	_, err := DecodeRaws(brokenToml)
	require.ErrorIs(t, err, errTOMLDecode)
	var parseErr toml.ParseError
	require.ErrorAs(t, err, &parseErr)
}

func TestTileSpawn_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	got := map[string]rawschema.TileSpawn{}
	for _, tl := range PtrSlice(raws.Tiles) {
		if tl.Spawn != nil {
			got[tl.Id] = *tl.Spawn
		}
	}
	want := map[string]rawschema.TileSpawn{
		"dirt":             {Autotile: true},
		"dirt_orange":      {Autotile: true},
		"dirt_tundra":      {Autotile: true},
		"dirt_peat":        {Autotile: true},
		"dirt_ash":         {Autotile: true},
		"crater_ground":    {Autotile: true},
		"scorched_ground":  {Autotile: true},
		"floor":            {Autotile: true},
		"road":             {Autotile: true},
		"grass_green":      {Autotile: true},
		"dwall":            {Autotile: true},
		"brick_cyan":       {Autotile: true},
		"cobble_teal":      {Autotile: true},
		"cobble_gold":      {Autotile: true},
		"brick_gray":       {Autotile: true},
		"cobble_cream":     {Autotile: true},
		"brick_blue":       {Autotile: true},
		"grass_olive":      {Autotile: true},
		"cobble_red":       {Autotile: true},
		"brick_navy":       {Autotile: true},
		"cobble_purple":    {Autotile: true},
		"cobble_slate":     {Autotile: true},
		"sand_pink":        {Autotile: true},
		"brick_amber":      {Autotile: true},
		"smooth_cream":     {Autotile: true},
		"sand_orange":      {Autotile: true},
		"brick_terracotta": {Autotile: true},
		"smooth_white":     {Autotile: true},
		"sand_red":         {Autotile: true},
		"brick_beige":      {Autotile: true},
		"smooth_teal":      {Autotile: true},
		"sand_slate":       {Autotile: true},
		"brick_rust":       {Autotile: true},
		"smooth_bluegray":  {Autotile: true},
		"plank_orange":     {Autotile: true},
		"plank_brown":      {Autotile: true},
		"plank_navy":       {Autotile: true},
		"plank_blue":       {Autotile: true},
		"wall":             {Tile: new("dwall"), Autotile: true},
		"void":             {Autotile: false},
	}
	for _, id := range wallMaterialTiles() {
		want[id] = rawschema.TileSpawn{Autotile: true}
	}
	assert.Equal(t, want, got)
}

func TestTileShelter_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	got := map[string]rawschema.Shelter{}
	tiles := PtrSlice(raws.Tiles)
	for i := range tiles {
		if s := TileShelter(tiles[i]); s != rawschema.Outdoor {
			got[tiles[i].Id] = s
		}
	}
	assert.Equal(t, map[string]rawschema.Shelter{
		"floor":           rawschema.Indoor,
		"floor_concrete":  rawschema.Indoor,
		"floor_linoleum":  rawschema.Indoor,
		"floor_tatami":    rawschema.Indoor,
		"floor_tile":      rawschema.Indoor,
		"floor_wood":      rawschema.Indoor,
		"plank_blue":      rawschema.Indoor,
		"plank_brown":     rawschema.Indoor,
		"plank_navy":      rawschema.Indoor,
		"plank_orange":    rawschema.Indoor,
		"smooth_bluegray": rawschema.Indoor,
		"smooth_cream":    rawschema.Indoor,
		"smooth_teal":     rawschema.Indoor,
		"smooth_white":    rawschema.Indoor,
	}, got)
}

func TestTileEnvironmentOf(t *testing.T) {
	t.Parallel()

	t.Run("FoliageとSnowMeltPercentを省くとゼロ値", func(t *testing.T) {
		t.Parallel()
		got := TileEnvironmentOf(rawschema.Tile{})
		assert.Equal(t, gc.TileEnvironment{Shelter: rawschema.Outdoor}, got)
	})

	t.Run("FoliageとSnowMeltPercentを指定すると反映する", func(t *testing.T) {
		t.Parallel()
		indoor := rawschema.Indoor
		foliage := rawschema.FoliageForest
		snowMelt := 50
		got := TileEnvironmentOf(rawschema.Tile{Shelter: &indoor, Foliage: &foliage, SnowMeltPercent: &snowMelt})
		assert.Equal(t, gc.TileEnvironment{Shelter: rawschema.Indoor, Foliage: rawschema.FoliageForest, SnowMeltPercent: 50}, got)
	})
}

func TestTiles_オートタイルの絵しか無いタイルは生成でオートタイルにする(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	exists := func(name string) bool {
		_, err := fs.Stat(assets.FS, "file/textures/tiles/"+name+"_.png")
		return err == nil
	}
	var missing []string
	for _, tl := range PtrSlice(raws.Tiles) {
		key := tl.SpriteRender.SpriteKey
		if exists(key+"_0") && !exists(key) && (tl.Spawn == nil || !tl.Spawn.Autotile) {
			missing = append(missing, tl.Id)
		}
	}
	assert.Empty(t, missing)
}

// wallMaterialTiles は DawnLike の壁のオートタイルの行の id を返す。素材ごとに明るさ4段
func wallMaterialTiles() []string {
	var ids []string
	for _, m := range []string{"stone", "rubble", "siding", "plaster", "pillar", "cobble", "board", "concrete", "wood", "brick", "reed", "ice"} {
		for n := 1; n <= 4; n++ {
			ids = append(ids, fmt.Sprintf("wall_%s_%d", m, n))
		}
	}
	return ids
}

func TestWallMaterialTiles_dwallと同じ描画順と通れなさと視界で並ぶ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	dwall, err := GetTile(raws, "dwall")
	require.NoError(t, err)
	type traits struct {
		depth                rawschema.SpriteDepth
		blockPass, blockView bool
	}
	want := traits{dwall.SpriteRender.Depth, dwall.BlockPass, dwall.BlockView}
	for _, id := range wallMaterialTiles() {
		tile, err := GetTile(raws, id)
		require.NoError(t, err, id)
		assert.Equal(t, want, traits{tile.SpriteRender.Depth, tile.BlockPass, tile.BlockView}, id)
	}
}

func TestDepthOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		depth rawschema.SpriteDepth
		want  gc.DepthNum
	}{
		{"床は床の段", rawschema.SpriteDepthFloor, gc.DepthNumFloor},
		{"床の模様は床の模様の段", rawschema.SpriteDepthDecal, gc.DepthNumDecal},
		{"床に置くものは床に置くものの段", rawschema.SpriteDepthRug, gc.DepthNumRug},
		{"高さのあるものは高さのあるものの段", rawschema.SpriteDepthTaller, gc.DepthNumTaller},
		{"描画順を持たない行は床に落ちる", "", gc.DepthNumFloor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, depthOf(tt.depth))
		})
	}
}
