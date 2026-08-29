package entityspec_test

import (
	"os"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
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

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Equal(t, []string{
		query.T(world, gc.AttackSword.Label),
		"Attack power", "25", "Accuracy", "80", "Hits", "2", "Attack cost", "100",
		"Element", query.T(world, gc.ElementTypeFire.String()),
	}, labels)
}

func TestUpdateSpec_無属性の近接武器は属性行を表示しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Melee.Add(e, &gc.Melee{
		Accuracy: 100, Damage: 8, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackFist, Cost: 50,
	})

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	// 無属性は Element 行が出ない
	assert.Equal(t, []string{
		query.T(world, gc.AttackFist.Label),
		"Attack power", "8", "Accuracy", "100", "Hits", "1", "Attack cost", "50",
	}, labels)
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

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Equal(t, []string{
		query.T(world, gc.AttackRifle.Label),
		"Attack power", "30", "Accuracy", "70", "Hits", "1", "Attack cost", "150",
		"Optimal range", "8", "Max range", "16", "Magazine", "3/5", "Reload", "20",
	}, labels)
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

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	// マガジンサイズ0は Magazine と Reload の行が出ない
	assert.Equal(t, []string{
		query.T(world, gc.AttackBow.Label),
		"Attack power", "30", "Accuracy", "70", "Hits", "1", "Attack cost", "80",
		"Optimal range", "4", "Max range", "10",
	}, labels)
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

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	// ゼロの装備ボーナス Sensation は行が出ない
	assert.Equal(t, []string{
		query.T(world, gc.EquipmentTorso.String()),
		"Defense", "+15", "Cold resist", "+3", "Heat resist", "+2",
		"Vitality", "+6", "Strength", "+1", "Dexterity", "+4", "Agility", "-1",
	}, labels)
}

func TestUpdateSpec_耐性のない防具は耐寒耐熱行を表示しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Wearable.Add(e, &gc.Wearable{
		Defense:           5,
		EquipmentCategory: gc.EquipmentHead,
	})

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	// 耐寒・耐熱0は行が出ない
	assert.Equal(t, []string{query.T(world, gc.EquipmentHead.String()), "Defense", "+5"}, labels)
}

func TestUpdateSpec_回復量は数値指定なら整数で表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.ProvidesHealing.Add(e, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 42})

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Equal(t, []string{"Vitality", "42"}, labels)
}

func TestUpdateSpec_回復量は割合指定ならパーセントで表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.ProvidesHealing.Add(e, &gc.ProvidesHealing{Kind: gc.HealRatio, Amount: 0.3})

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Equal(t, []string{"Vitality", "30%"}, labels)
}

func TestUpdateSpec_回復量は未知の種別ならハイフンで表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	// gc.HealAmountKind に現在定義されていない値を使い、default分岐を狙う
	const unknownKind = gc.HealAmountKind(99)
	world.Components.ProvidesHealing.Add(e, &gc.ProvidesHealing{Kind: unknownKind, Amount: 10})

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	// 未知の種別はハイフン表示にフォールバックする
	assert.Equal(t, []string{"Vitality", "-"}, labels)
}

func TestUpdateSpec_栄養と価値と重量を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.ProvidesNutrition.Add(e, &gc.ProvidesNutrition{Amount: 25})
	world.Components.Value.Add(e, &gc.Value{Value: 1200})
	world.Components.Weight.Add(e, &gc.Weight{})

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Equal(t, []string{"Nutrition", "25", "Value", consts.Currency(1200).String(), "Weight", "0㎎"}, labels)
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

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	assert.Equal(t, []string{
		"Book", "Skill", query.T(world, gc.SkillName(gc.SkillSword)),
		"Lv", "2 " + consts.IconArrowRight + " 5", "Progress", "30%",
	}, labels)
}

func TestUpdateSpec_進捗が0の本は進捗行を表示しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e := world.ECS.NewEntity()
	world.Components.Book.Add(e, &gc.Book{
		Effort: gc.IntPool{Current: 0, Max: 0},
	})

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRows(world, e), nil))

	// 工数未設定の Progress 行とスキル未設定の Skill 行は出ない
	assert.Equal(t, []string{"Book"}, labels)
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

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRowsFromSpec(world, spec), nil))

	assert.Equal(t, []string{
		query.T(world, gc.AttackSpear.Label),
		"Attack power", "12", "Accuracy", "90", "Hits", "1", "Attack cost", "80",
		"Element", query.T(world, gc.ElementTypeThunder.String()),
	}, labels)
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

	labels := uicore.CollectLabels(entityspec.BuildSpecPanel(entityspec.SpecRowsFromSpec(world, spec), nil))

	// Fire・Wearable・Healing・Nutrition・Book・Value・Weight の行がこの並びで連結される
	assert.Equal(t, []string{
		query.T(world, gc.AttackRifle.Label),
		"Attack power", "30", "Accuracy", "70", "Hits", "1", "Attack cost", "150",
		"Optimal range", "8", "Max range", "16", "Magazine", "3/5", "Reload", "20",
		query.T(world, gc.EquipmentTorso.String()),
		"Defense", "+15", "Cold resist", "+3",
		"Vitality", "42",
		"Nutrition", "25",
		"Book", "Progress", "30%",
		"Value", consts.Currency(1200).String(),
		"Weight", "0㎎",
	}, labels)
}
