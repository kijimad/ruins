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

func TestSaveLoadEffectComponents(t *testing.T) {
	t.Parallel()
	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "save_test_")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// ワールドを作成
	w := testutil.InitTestWorld(t)

	// 回復アイテムを作成
	healingItem := w.Manager.NewEntity()
	healingItem.AddComponent(w.Components.Item, &gc.Item{})
	healingItem.AddComponent(w.Components.Name, &gc.Name{Name: "回復薬"})
	healingItem.AddComponent(w.Components.Consumable, &gc.Consumable{
		UsableScene: gc.UsableSceneAny,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupAlly,
			TargetNum:   gc.TargetSingle,
		},
	})
	healingItem.AddComponent(w.Components.ProvidesHealing, &gc.ProvidesHealing{
		Amount: gc.RatioAmount{Ratio: 0.5}, // HP50%回復
	})

	// ダメージアイテムを作成
	damageItem := w.Manager.NewEntity()
	damageItem.AddComponent(w.Components.Item, &gc.Item{})
	damageItem.AddComponent(w.Components.Name, &gc.Name{Name: "手榴弾"})
	damageItem.AddComponent(w.Components.Consumable, &gc.Consumable{
		UsableScene: gc.UsableSceneBattle,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupEnemy,
			TargetNum:   gc.TargetAll,
		},
	})
	damageItem.AddComponent(w.Components.InflictsDamage, &gc.InflictsDamage{
		Amount: 30, // 30ダメージ
	})

	// 両方の効果を持つアイテムを作成（架空）
	mixedItem := w.Manager.NewEntity()
	mixedItem.AddComponent(w.Components.Item, &gc.Item{})
	mixedItem.AddComponent(w.Components.Name, &gc.Name{Name: "特殊アイテム"})
	mixedItem.AddComponent(w.Components.ProvidesHealing, &gc.ProvidesHealing{
		Amount: gc.NumeralAmount{Numeral: 20}, // 20HP回復
	})
	mixedItem.AddComponent(w.Components.InflictsDamage, &gc.InflictsDamage{
		Amount: 15, // 15ダメージ
	})

	// セーブマネージャーを作成
	sm := NewSerializationManager(tempDir)

	// ワールドを保存
	err = sm.SaveWorld(w, "test_effect")
	require.NoError(t, err)

	// セーブファイルが存在することを確認
	saveFile := filepath.Join(tempDir, "test_effect.json")
	_, err = os.Stat(saveFile)
	require.NoError(t, err)

	// 新しいワールドを作成してロード
	newWorld := testutil.InitTestWorld(t)

	err = sm.LoadWorld(newWorld, "test_effect")
	require.NoError(t, err)

	// 回復アイテムが正しくロードされたことを確認
	healingCount := 0
	newWorld.Manager.Join(
		newWorld.Components.Item,
		newWorld.Components.ProvidesHealing,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		healing := newWorld.Components.ProvidesHealing.Get(entity).(*gc.ProvidesHealing)

		switch name.Name {
		case "回復薬":
			// Amount インターフェースの型を確認
			ratioAmount, ok := healing.Amount.(gc.RatioAmount)
			assert.True(t, ok, "回復薬のAmountはRatioAmountでなければならない")
			assert.Equal(t, 0.5, ratioAmount.Ratio)
			healingCount++
		case "特殊アイテム":
			// Amount インターフェースの型を確認
			numeralAmount, ok := healing.Amount.(gc.NumeralAmount)
			assert.True(t, ok, "特殊アイテムのAmountはNumeralAmountでなければならない")
			assert.Equal(t, 20, numeralAmount.Numeral)
			healingCount++
		}
	}))
	assert.Equal(t, 0, healingCount, "フィールドアイテムは保存されない（バックパックと装備アイテムのみ保存）")

	// ダメージアイテムが正しくロードされたことを確認
	damageCount := 0
	newWorld.Manager.Join(
		newWorld.Components.Item,
		newWorld.Components.InflictsDamage,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		damage := newWorld.Components.InflictsDamage.Get(entity).(*gc.InflictsDamage)

		switch name.Name {
		case "手榴弾":
			assert.Equal(t, 30, damage.Amount)
			damageCount++
		case "特殊アイテム":
			assert.Equal(t, 15, damage.Amount)
			damageCount++
		}
	}))
	assert.Equal(t, 0, damageCount, "フィールドアイテムは保存されない（バックパックと装備アイテムのみ保存）")

	// Consumableコンポーネントも正しくロードされることを確認
	consumableCount := 0
	newWorld.Manager.Join(
		newWorld.Components.Item,
		newWorld.Components.Consumable,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		consumable := newWorld.Components.Consumable.Get(entity).(*gc.Consumable)

		switch name.Name {
		case "回復薬":
			assert.Equal(t, gc.UsableSceneAny, consumable.UsableScene)
			assert.Equal(t, gc.TargetGroupAlly, consumable.TargetType.TargetGroup)
			assert.Equal(t, gc.TargetSingle, consumable.TargetType.TargetNum)
			consumableCount++
		case "手榴弾":
			assert.Equal(t, gc.UsableSceneBattle, consumable.UsableScene)
			assert.Equal(t, gc.TargetGroupEnemy, consumable.TargetType.TargetGroup)
			assert.Equal(t, gc.TargetAll, consumable.TargetType.TargetNum)
			consumableCount++
		}
	}))
	assert.Equal(t, 0, consumableCount, "フィールドアイテムは保存されない（バックパックと装備アイテムのみ保存）")
}
