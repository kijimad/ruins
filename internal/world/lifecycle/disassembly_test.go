package lifecycle_test

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollDisassemblyYields(t *testing.T) {
	t.Parallel()

	t.Run("確定枠は通常分解で必ず出る", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(1, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Prying,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "2"}},
		}

		for range 100 {
			stacks, err := lifecycle.RollDisassemblyYields(rng, def, 0, 1, true)
			require.NoError(t, err)
			require.Len(t, stacks, 1)
			assert.Equal(t, lifecycle.YieldStack{Name: "鉄くず", Count: 2}, stacks[0])
		}
	})

	t.Run("amountMax指定は範囲内の個数になる", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(2, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Prying,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "硬木", Count: "1d3"}},
		}

		seen := map[int]bool{}
		for range 200 {
			stacks, err := lifecycle.RollDisassemblyYields(rng, def, 0, 1, true)
			require.NoError(t, err)
			require.Len(t, stacks, 1)
			assert.GreaterOrEqual(t, stacks[0].Count, 1)
			assert.LessOrEqual(t, stacks[0].Count, 3)
			seen[stacks[0].Count] = true
		}
		assert.Equal(t, map[int]bool{1: true, 2: true, 3: true}, seen, "1..3のすべての個数が出るべき")
	})

	t.Run("確率枠はスキルとグレード補正で100に達すると必ず出る", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(3, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Precision,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "ネジ", Count: "1", Chance: new(oapi.DisassemblyChance(60))}},
		}

		// chance60 + skill30 + (grade2-1)*10 = 100
		for range 100 {
			stacks, err := lifecycle.RollDisassemblyYields(rng, def, 30, 2, true)
			require.NoError(t, err)
			require.Len(t, stacks, 1)
		}
	})

	t.Run("確率枠は補正なしだとおおむね確率どおりに出る", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(4, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Precision,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "ネジ", Count: "1", Chance: new(oapi.DisassemblyChance(50))}},
		}

		hit := 0
		for range 2000 {
			stacks, err := lifecycle.RollDisassemblyYields(rng, def, 0, 1, true)
			require.NoError(t, err)
			if len(stacks) == 1 {
				hit++
			}
		}
		assert.InDelta(t, 1000, hit, 150, "50%前後で当選するべき")
	})

	t.Run("ボーナス枠はminSkill条件を満たすと出る", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(5, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Prying,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1"}},
			Bonus:        &[]oapi.DisassemblyBonus{{Name: "鉄", Count: "1", MinSkill: new(oapi.SkillLevel(10))}},
		}

		low, err := lifecycle.RollDisassemblyYields(rng, def, 9, 1, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}}, low, "スキル9ではボーナスが出ないべき")

		high, err := lifecycle.RollDisassemblyYields(rng, def, 10, 1, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}, {Name: "鉄", Count: 1}}, high)
	})

	t.Run("ボーナス枠はminGrade条件を満たすと出る", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(6, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Prying,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1"}},
			Bonus:        &[]oapi.DisassemblyBonus{{Name: "鉄", Count: "1", MinGrade: new(oapi.ToolGrade(2))}},
		}

		low, err := lifecycle.RollDisassemblyYields(rng, def, 0, 1, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}}, low, "グレード1ではボーナスが出ないべき")

		high, err := lifecycle.RollDisassemblyYields(rng, def, 0, 2, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}, {Name: "鉄", Count: 1}}, high)
	})

	t.Run("ボーナス枠は両方指定なら両方満たす必要がある", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(8, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Prying,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1"}},
			Bonus: &[]oapi.DisassemblyBonus{
				{Name: "鉄", Count: "1", MinSkill: new(oapi.SkillLevel(10)), MinGrade: new(oapi.ToolGrade(2))},
			},
		}

		skillOnly, err := lifecycle.RollDisassemblyYields(rng, def, 10, 1, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}}, skillOnly, "スキルだけ満たしても出ないべき")

		gradeOnly, err := lifecycle.RollDisassemblyYields(rng, def, 0, 2, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}}, gradeOnly, "グレードだけ満たしても出ないべき")

		both, err := lifecycle.RollDisassemblyYields(rng, def, 10, 2, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}, {Name: "鉄", Count: 1}}, both)
	})

	t.Run("ボーナス枠は条件が両方未指定なら出さない", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(9, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Prying,
			BaseAP:       100,
			Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1"}},
			Bonus:        &[]oapi.DisassemblyBonus{{Name: "鉄", Count: "1"}},
		}

		stacks, err := lifecycle.RollDisassemblyYields(rng, def, 100, 3, true)

		require.NoError(t, err)
		assert.Equal(t, []lifecycle.YieldStack{{Name: "鉄くず", Count: 1}}, stacks, "無条件のボーナス定義は無効として扱うべき")
	})

	t.Run("破壊回収は確定枠のみを低確率で出す", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(7, 0))
		def := &oapi.Disassembly{
			ToolCategory: oapi.Prying,
			BaseAP:       100,
			Yields: []oapi.DisassemblyYield{
				{Name: "鉄くず", Count: "1"},
				{Name: "ネジ", Count: "1", Chance: new(oapi.DisassemblyChance(100))},
			},
			Bonus: &[]oapi.DisassemblyBonus{{Name: "鉄", Count: "1", MinSkill: new(oapi.SkillLevel(0))}},
		}

		hit := 0
		for range 2000 {
			stacks, err := lifecycle.RollDisassemblyYields(rng, def, 100, 3, false)
			require.NoError(t, err)
			for _, s := range stacks {
				assert.Equal(t, "鉄くず", s.Name, "破壊回収では確定枠以外が出ないべき")
			}
			if len(stacks) > 0 {
				hit++
			}
		}
		assert.InDelta(t, 600, hit, 150, "確定枠はDestroySalvageChance前後で当選するべき")
	})
}
