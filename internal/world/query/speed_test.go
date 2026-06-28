package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

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
		entity.AddComponent(world.Components.Hunger, &gc.Hunger{Current: 20, Max: 100}) // 飢餓状態

		speed := CalculateSpeed(world, entity)
		// 基本100 - 飢餓ペナルティ50 = 50
		assert.Equal(t, 50, speed)
	})

	t.Run("過積載によるペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 100, Current: 150}) // 50%超過

		speed := CalculateSpeed(world, entity)
		// 基本100 - 超過ペナルティ(50*25/100=12) = 88
		assert.Equal(t, 88, speed)
	})

	t.Run("体温異常によるペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// lifecycleに依存しないよう手動でエンティティを構築する
		abils := &gc.Abilities{
			Vitality:  gc.Ability{Base: 10, Total: 10},
			Strength:  gc.Ability{Base: 8, Total: 8},
			Sensation: gc.Ability{Base: 7, Total: 7},
			Dexterity: gc.Ability{Base: 6, Total: 6},
			Agility:   gc.Ability{Base: 9, Total: 9},
			Defense:   gc.Ability{Base: 5, Total: 5},
		}
		skills := gc.NewSkills()
		hs := &gc.HealthStatus{}

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Player, &gc.Player{})
		entity.AddComponent(world.Components.Abilities, abils)
		entity.AddComponent(world.Components.Skills, skills)
		entity.AddComponent(world.Components.HealthStatus, hs)
		entity.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, hs))

		// 通常時のSpeedを記録
		normalSpeed := CalculateSpeed(world, entity)

		// 低体温を設定してCharModifiersを再計算
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityMedium,
		})
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
		entity.AddComponent(world.Components.Hunger, &gc.Hunger{Current: 5, Max: 100})                   // 餓死寸前(-75)
		entity.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 100, Current: 400}) // 大幅超過（最大-75）

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
		entity.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 100, Current: 80})

		penalty := calculateOverweightPenalty(world, entity)
		assert.Equal(t, 0, penalty)
	})

	t.Run("50%超過", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 100, Current: 150})

		penalty := calculateOverweightPenalty(world, entity)
		// 50 * 25 / 100 = 12.5 -> -12
		assert.Equal(t, -12, penalty)
	})

	t.Run("最大ペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 100, Current: 500}) // 400%超過

		penalty := calculateOverweightPenalty(world, entity)
		// 最大-75
		assert.Equal(t, -75, penalty)
	})
}
