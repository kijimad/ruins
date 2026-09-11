package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// TestSourceLabel は熟練度の要因からラベルを組む純関数を検証する。
// 出力は query.T の翻訳を挟むため、書式の不変部分を Contains で固定する。
// world は Ark が並行安全でないため各サブテストで作る。
func TestSourceLabel(t *testing.T) {
	t.Parallel()

	t.Run("スキルはLvと量を書式へ含める", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		got := sourceLabel(world, gc.ProficiencySource{Kind: gc.SourceSkill, Skill: gc.SkillSword, Amount: 5})
		assert.Contains(t, got, "Lv5")
	})

	t.Run("能力値は量を書式へ含める", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		got := sourceLabel(world, gc.ProficiencySource{Kind: gc.SourceAbility, Ability: gc.AblSTR, Amount: 3})
		assert.Contains(t, got, "3")
	})

	t.Run("身体機能はパーセントと量を書式へ含める", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		got := sourceLabel(world, gc.ProficiencySource{Kind: gc.SourceBodyFunc, BodyFunc: gc.BodyFuncManipulation, Amount: 80})
		assert.Contains(t, got, "80%")
	})

	t.Run("疲労と空腹と睡眠は空でないラベルを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.NotEmpty(t, sourceLabel(world, gc.ProficiencySource{Kind: gc.SourceFatigue, Fatigue: gc.FatigueTired}))
		assert.NotEmpty(t, sourceLabel(world, gc.ProficiencySource{Kind: gc.SourceHunger, Hunger: gc.HungerSatiated}))
		assert.NotEmpty(t, sourceLabel(world, gc.ProficiencySource{Kind: gc.SourceSleeping}))
	})

	t.Run("未知の種別はパニックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.Panics(t, func() {
			_ = sourceLabel(world, gc.ProficiencySource{Kind: gc.ProficiencySourceKind("unknown")})
		})
	})
}
