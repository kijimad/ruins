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

		entity := world.ECS.NewEntity()

		speed := CalculateSpeed(world, entity)
		// 基本値100、能力値なし
		assert.Equal(t, 100, speed)
	})

	t.Run("能力値によるボーナス", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.Abilities.Add(entity, &gc.Abilities{
			Agility:   gc.Ability{Total: 10},
			Dexterity: gc.Ability{Total: 5},
		})

		speed := CalculateSpeed(world, entity)
		// 基本100 + AGI*2 (20) + DEX*1 (5) = 125
		assert.Equal(t, 125, speed)
	})

	t.Run("飢餓の栄養失調は速度を下げる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		// 飢餓の Malnutrition 不調は全身性 8*3=24 を意識へ落とし、歩行76%が速度へ乗る
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetGaugeCondition(gc.ConditionMalnutrition, gc.SeveritySevere)
		world.Components.HealthStatus.Add(entity, hs)

		speed := CalculateSpeed(world, entity)
		// 素の速度100 × 歩行76% = 76
		assert.Equal(t, 76, speed)
	})

	t.Run("能力の素速度に栄養失調が乗算で乗る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.Abilities.Add(entity, &gc.Abilities{
			Agility:   gc.Ability{Total: 10},
			Dexterity: gc.Ability{Total: 5},
		})
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetGaugeCondition(gc.ConditionMalnutrition, gc.SeveritySevere)
		world.Components.HealthStatus.Add(entity, hs)

		speed := CalculateSpeed(world, entity)
		// 素の速度 100 + AGI*2(20) + DEX*1(5) = 125。歩行76%が乗算で掛かり 125*76% = 95
		assert.Equal(t, 95, speed)
	})

	t.Run("過積載によるペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: 100, Current: 150}) // 50%超過

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

		entity := world.ECS.NewEntity()
		world.Components.Player.Add(entity, &gc.Player{})
		world.Components.Abilities.Add(entity, abils)
		world.Components.Skills.Add(entity, skills)
		world.Components.HealthStatus.Add(entity, hs)

		// 通常時のSpeedを記録
		normalSpeed := CalculateSpeed(world, entity)

		// 低体温を付ける。Add で hs は格納先へコピーされ別物になるので、
		// ローカルの hs でなく Get で格納先を取り直して書き換える
		world.Components.HealthStatus.Get(entity).Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityMedium,
		})

		coldSpeed := CalculateSpeed(world, entity)
		t.Logf("normalSpeed=%d coldSpeed=%d", normalSpeed, coldSpeed)
		assert.Less(t, coldSpeed, normalSpeed, "低体温によりSpeedが低下するべき")
	})

	t.Run("複合ペナルティで最小値に達する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 5, Max: 100})                   // 飢餓(-20)
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: 100, Current: 400}) // 大幅超過（最大-75）

		speed := CalculateSpeed(world, entity)
		// ペナルティが大きくても最小値25を下回らない
		assert.Equal(t, 25, speed)
	})
}

func TestOverweightPenalty(t *testing.T) {
	t.Parallel()

	t.Run("超過なし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: 100, Current: 80})

		penalty := calculateOverweightPenalty(world, entity)
		assert.Equal(t, 0, penalty)
	})

	t.Run("50%超過", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: 100, Current: 150})

		penalty := calculateOverweightPenalty(world, entity)
		// 50 * 25 / 100 = 12.5 -> -12
		assert.Equal(t, -12, penalty)
	})

	t.Run("最大ペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: 100, Current: 500}) // 400%超過

		penalty := calculateOverweightPenalty(world, entity)
		// 最大-75
		assert.Equal(t, -75, penalty)
	})
}
