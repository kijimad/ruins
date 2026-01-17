package save

import (
	"os"
	"path/filepath"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestSaveLoadItemLocations(t *testing.T) {
	t.Parallel()
	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "save_test_")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// ワールドを作成
	w := testutil.InitTestWorld(t)

	// アイテムエンティティを作成してバックパックに追加
	item1 := w.Manager.NewEntity()
	item1.AddComponent(w.Components.Item, &gc.Item{})
	item1.AddComponent(w.Components.Name, &gc.Name{Name: "テストアイテム1"})
	item1.AddComponent(w.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

	// アイテムエンティティを作成してフィールドに配置
	item2 := w.Manager.NewEntity()
	item2.AddComponent(w.Components.Item, &gc.Item{})
	item2.AddComponent(w.Components.Name, &gc.Name{Name: "テストアイテム2"})
	item2.AddComponent(w.Components.ItemLocationOnField, &gc.LocationOnField{})

	// キャラクターエンティティを作成
	character := w.Manager.NewEntity()
	character.AddComponent(w.Components.Name, &gc.Name{Name: "テストキャラ"})
	character.AddComponent(w.Components.FactionAlly, &gc.FactionAllyData{})
	character.AddComponent(w.Components.Player, &gc.Player{})

	// アイテムエンティティを作成して装備
	item3 := w.Manager.NewEntity()
	item3.AddComponent(w.Components.Item, &gc.Item{})
	item3.AddComponent(w.Components.Name, &gc.Name{Name: "テスト装備"})
	item3.AddComponent(w.Components.ItemLocationEquipped, &gc.LocationEquipped{
		Owner:         character,
		EquipmentSlot: gc.EquipmentSlotNumber(0), // メインハンドに相当
	})
	item3.AddComponent(w.Components.EquipmentChanged, &gc.EquipmentChanged{})

	// セーブマネージャーを作成
	sm := NewSerializationManager(tempDir)

	// ワールドを保存
	err = sm.SaveWorld(w, "test_slot")
	require.NoError(t, err)

	// セーブファイルが存在することを確認
	saveFile := filepath.Join(tempDir, "test_slot.json")
	_, err = os.Stat(saveFile)
	require.NoError(t, err)

	// 新しいワールドを作成してロード
	newWorld := testutil.InitTestWorld(t)

	err = sm.LoadWorld(newWorld, "test_slot")
	require.NoError(t, err)

	// アイテムがバックパックに存在することを確認
	backpackItemCount := 0
	newWorld.Manager.Join(
		newWorld.Components.Item,
		newWorld.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		assert.Equal(t, "テストアイテム1", name.Name)
		backpackItemCount++
	}))
	assert.Equal(t, 1, backpackItemCount, "バックパックのアイテムが正しくロードされていない")

	// アイテムがフィールドに存在することを確認
	fieldItemCount := 0
	newWorld.Manager.Join(
		newWorld.Components.Item,
		newWorld.Components.ItemLocationOnField,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		assert.Equal(t, "テストアイテム2", name.Name)
		fieldItemCount++
	}))
	assert.Equal(t, 0, fieldItemCount, "フィールドアイテムは保存されない（バックパックと装備アイテムのみ保存）")

	// 装備アイテムが存在することを確認
	equippedItemCount := 0
	var loadedOwner ecs.Entity
	newWorld.Manager.Join(
		newWorld.Components.Item,
		newWorld.Components.ItemLocationEquipped,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		assert.Equal(t, "テスト装備", name.Name)

		// 装備情報を確認
		equipped := newWorld.Components.ItemLocationEquipped.Get(entity).(*gc.LocationEquipped)
		assert.Equal(t, gc.EquipmentSlotNumber(0), equipped.EquipmentSlot)

		// Owner参照が復元されていることを確認
		assert.True(t, equipped.Owner.HasComponent(newWorld.Components.Name),
			"Owner参照が無効なエンティティを指している: equipped.Owner = %d", equipped.Owner)
		loadedOwner = equipped.Owner

		equippedItemCount++
	}))
	assert.Equal(t, 1, equippedItemCount, "装備アイテムが正しくロードされていない")

	// キャラクターが存在することを確認
	characterCount := 0
	newWorld.Manager.Join(
		newWorld.Components.FactionAlly,
		newWorld.Components.Player,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		assert.Equal(t, "テストキャラ", name.Name)

		// この"キャラクター"が装備のOwnerとして正しく参照されていることを確認
		if loadedOwner != 0 {
			assert.Equal(t, entity, loadedOwner, "装備のOwner参照が正しいキャラクターを指していない")
		}

		characterCount++
	}))
	assert.Equal(t, 1, characterCount, "キャラクターが正しくロードされていない")
}
