package entityspec_test

import (
	"os"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// UIResources のロードや widget.Text の生成が ebiten の実行状態に依存するため必要
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestUpdateSpec_近接武器の攻撃性能を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Melee.Add(e, &gc.Melee{
		Accuracy: 80, Damage: 25, AttackCount: 2,
		Element: gc.ElementTypeFire, AttackCategory: gc.AttackSword, Cost: 100,
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, query.T(world, gc.AttackSword.Label), "武器種別ラベルが表示される")
	assert.Contains(t, labels, "25", "攻撃力の値が表示される")
	assert.Contains(t, labels, "80", "命中率の値が表示される")
	assert.Contains(t, labels, "2", "攻撃回数の値が表示される")
	assert.Contains(t, labels, "100", "コストの値が表示される")
	assert.Contains(t, labels, query.T(world, gc.ElementTypeFire.String()), "属性名が表示される")
}

func TestUpdateSpec_無属性の近接武器は属性行を表示しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Melee.Add(e, &gc.Melee{
		Accuracy: 100, Damage: 8, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackFist, Cost: 50,
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.NotContains(t, labels, "Element", "無属性の場合は属性行が表示されない")
}

func TestUpdateSpec_マガジンのある火器は弾数と射程を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	// AttackRifle は enum.go の rangeParams に登録済みなので、射程行が表示される前提が成り立つ
	world.Components.Fire.Add(e, &gc.Fire{
		Accuracy: 70, Damage: 30, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackRifle, Cost: 150,
		Magazine: 3, MagazineSize: 5, ReloadEffort: 20,
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "Optimal range", "適正射程ラベルが表示される")
	assert.Contains(t, labels, "Max range", "最大射程ラベルが表示される")
	assert.Contains(t, labels, "Magazine", "弾数ラベルが表示される")
	assert.Contains(t, labels, "3/5", "現在弾数/最大弾数が表示される")
	assert.Contains(t, labels, "Reload", "装填ラベルが表示される")
	assert.Contains(t, labels, "20", "リロード工数が表示される")
}

func TestUpdateSpec_マガジンサイズ0の火器は弾数を表示しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Fire.Add(e, &gc.Fire{
		Accuracy: 70, Damage: 30, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackBow, Cost: 80,
		MagazineSize: 0,
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.NotContains(t, labels, "Magazine", "マガジンサイズが0の場合は弾数行が表示されない")
	assert.NotContains(t, labels, "Reload", "マガジンサイズが0の場合は装填行が表示されない")
}

func TestUpdateSpec_防具は防御力と耐性を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Wearable.Add(e, &gc.Wearable{
		Defense:           15,
		EquipmentCategory: gc.EquipmentTorso,
		InsulationCold:    3,
		InsulationHeat:    2,
		// 各項目の値をすべて別々にし、"+N" の一致がどの行由来か一意に特定できるようにする
		EquipBonus: gc.EquipBonus{
			Vitality: 6, Strength: 1, Sensation: 0, Dexterity: 4, Agility: -1,
		},
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "+15", "防御力が符号付きで表示される")
	assert.Contains(t, labels, "Cold resist", "耐寒ラベルが表示される")
	assert.Contains(t, labels, "+3", "耐寒値が表示される")
	assert.Contains(t, labels, "Heat resist", "耐熱ラベルが表示される")
	assert.Contains(t, labels, "+2", "耐熱値が表示される")
	assert.Contains(t, labels, "+6", "体力ボーナスが表示される")
	assert.Contains(t, labels, "+1", "筋力ボーナスが表示される")
	assert.Contains(t, labels, "+4", "器用ボーナスが表示される")
	assert.Contains(t, labels, "-1", "敏捷ボーナスが負値でも表示される")
	assert.NotContains(t, labels, "Sensation", "ゼロの装備ボーナスは表示されない")
}

func TestUpdateSpec_耐性のない防具は耐寒耐熱行を表示しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Wearable.Add(e, &gc.Wearable{
		Defense:           5,
		EquipmentCategory: gc.EquipmentHead,
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.NotContains(t, labels, "Cold resist", "耐寒0の場合は行が表示されない")
	assert.NotContains(t, labels, "Heat resist", "耐熱0の場合は行が表示されない")
}

func TestUpdateSpec_回復量は数値指定なら整数で表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.ProvidesHealing.Add(e, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 42})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "Vitality", "回復量ラベルが表示される")
	assert.Contains(t, labels, "42", "絶対量がそのまま表示される")
}

func TestUpdateSpec_回復量は割合指定ならパーセントで表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.ProvidesHealing.Add(e, &gc.ProvidesHealing{Kind: gc.HealRatio, Amount: 0.3})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "30%", "割合が百分率表示される")
}

func TestUpdateSpec_回復量は未知の種別ならハイフンで表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	// gc.HealAmountKind に現在定義されていない値を使い、default分岐を狙う
	const unknownKind = gc.HealAmountKind(99)
	world.Components.ProvidesHealing.Add(e, &gc.ProvidesHealing{Kind: unknownKind, Amount: 10})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "-", "未知の種別はハイフン表示にフォールバックする")
}

func TestUpdateSpec_栄養と価値と重量を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.ProvidesNutrition.Add(e, &gc.ProvidesNutrition{Amount: 25})
	world.Components.Value.Add(e, &gc.Value{Value: 1200})
	world.Components.Weight.Add(e, &gc.Weight{})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "Nutrition", "栄養ラベルが表示される")
	assert.Contains(t, labels, "25", "栄養量が表示される")
	assert.Contains(t, labels, "Value", "価値ラベルが表示される")
	assert.Contains(t, labels, consts.Currency(1200).String(), "価値がカンマ区切りの通貨表記で表示される")
	assert.Contains(t, labels, "Weight", "重量ラベルが表示される")
}

func TestUpdateSpec_本はスキル情報と進捗を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Book.Add(e, &gc.Book{
		Effort: gc.IntPool{Current: 30, Max: 100},
		Skill: &gc.SkillBookEffect{
			TargetSkill:   gc.SkillSword,
			RequiredLevel: 2,
			MaxLevel:      5,
		},
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "Book", "本ヘッダーが表示される")
	assert.Contains(t, labels, "Skill", "スキルラベルが表示される")
	assert.Contains(t, labels, query.T(world, gc.SkillName(gc.SkillSword)), "対象スキル名が表示される")
	assert.Contains(t, labels, "Lv", "レベルラベルが表示される")
	assert.Contains(t, labels, "Progress", "進捗ラベルが表示される")
	assert.Contains(t, labels, "30%", "現在工数から進捗率が計算される")
}

func TestUpdateSpec_進捗が0の本は進捗行を表示しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Book.Add(e, &gc.Book{
		Effort: gc.IntPool{Current: 0, Max: 0},
	})

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Contains(t, labels, "Book", "本ヘッダーは表示される")
	assert.NotContains(t, labels, "Progress", "工数が未設定の場合は進捗行が表示されない")
	assert.NotContains(t, labels, "Skill", "スキル効果未設定の場合はスキル行が表示されない")
}

func TestUpdateSpecFromSpec_エンティティを生成せずに近接武器の性能を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	spec := gc.EntitySpec{
		Melee: &gc.Melee{
			Accuracy: 90, Damage: 12, AttackCount: 1,
			Element: gc.ElementTypeThunder, AttackCategory: gc.AttackSpear, Cost: 80,
		},
	}

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRowsFromSpec(world, spec), nil))

	assert.Contains(t, labels, query.T(world, gc.AttackSpear.Label), "武器種別ラベルが表示される")
	assert.Contains(t, labels, "12", "攻撃力の値が表示される")
	assert.Contains(t, labels, query.T(world, gc.ElementTypeThunder.String()), "属性名が表示される")
}

func TestUpdateSpecFromSpec_エンティティを生成せずに複数コンポーネントを同時に表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	spec := gc.EntitySpec{
		Fire: &gc.Fire{
			Accuracy: 70, Damage: 30, AttackCount: 1,
			Element: gc.ElementTypeNone, AttackCategory: gc.AttackRifle, Cost: 150,
			Magazine: 3, MagazineSize: 5, ReloadEffort: 20,
		},
		Wearable: &gc.Wearable{
			Defense:           15,
			EquipmentCategory: gc.EquipmentTorso,
			InsulationCold:    3,
		},
		ProvidesHealing:   &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 42},
		ProvidesNutrition: &gc.ProvidesNutrition{Amount: 25},
		Book: &gc.Book{
			Effort: gc.IntPool{Current: 30, Max: 100},
		},
		Value:  &gc.Value{Value: 1200},
		Weight: &gc.Weight{},
	}

	labels := ui.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRowsFromSpec(world, spec), nil))

	assert.Contains(t, labels, "Magazine", "Fire由来の弾数行が表示される")
	assert.Contains(t, labels, "+15", "Wearable由来の防御力が表示される")
	assert.Contains(t, labels, "Cold resist", "Wearable由来の耐寒行が表示される")
	assert.Contains(t, labels, "42", "ProvidesHealing由来の回復量が表示される")
	assert.Contains(t, labels, "Nutrition", "ProvidesNutrition由来のラベルが表示される")
	assert.Contains(t, labels, "Progress", "Book由来の進捗行が表示される")
	assert.Contains(t, labels, consts.Currency(1200).String(), "Value由来の価値が表示される")
	assert.Contains(t, labels, "Weight", "Weight由来のラベルが表示される")
}
