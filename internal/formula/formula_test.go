package formula

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcHitRate(t *testing.T) {
	t.Parallel()
	// 器用度と敏捷度が同じ場合、武器命中率がそのまま反映される
	assert.Equal(t, 80, CalcHitRate(10, 10, 80))

	// 器用度が高いと命中率が上がる
	assert.Equal(t, 90, CalcHitRate(15, 10, 80))

	// 最大値でクランプされる
	assert.Equal(t, MaxHitRate, CalcHitRate(50, 10, 90))

	// 最小値でクランプされる
	assert.Equal(t, MinHitRate, CalcHitRate(1, 50, 50))
}

func TestClampHitRate(t *testing.T) {
	t.Parallel()
	assert.Equal(t, MaxHitRate, ClampHitRate(100))
	assert.Equal(t, MinHitRate, ClampHitRate(0))
	assert.Equal(t, 50, ClampHitRate(50))
}

func TestApplyCritical(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 15, ApplyCritical(10))
	assert.Equal(t, 0, ApplyCritical(0))
	assert.Equal(t, 1, ApplyCritical(1))
}

func TestCalcHP(t *testing.T) {
	t.Parallel()
	// HP = 30 + vitality*8 + strength + sensation
	assert.Equal(t, 30+10*8+5+5, CalcHP(10, 5, 5))
	assert.Equal(t, 30, CalcHP(0, 0, 0))
}

func TestDamage(t *testing.T) {
	t.Parallel()
	// 能力値10・ダイス3・武器5の和は18、防御は4
	base := DamageInput{Ability: 10, Die: 3, WeaponDamage: 5, SkillMult: 100, ElementMult: 100, Defense: 4}
	tests := []struct {
		name string
		edit func(*DamageInput)
		want int
	}{
		{"和から防御を引く", func(*DamageInput) {}, 14},
		{"加算補正は和に足す", func(in *DamageInput) { in.Modifier = 2 }, 16},
		{"熟練の倍率は和に切り捨てで掛ける", func(in *DamageInput) { in.SkillMult = 150 }, 23},
		{"クリティカルは1.5倍", func(in *DamageInput) { in.Critical = true }, 23},
		{"属性耐性は防御の前に切り捨てで掛ける", func(in *DamageInput) { in.ElementMult = 85 }, 11},
		{"属性耐性が負でも最低保証を経て防御を引く", func(in *DamageInput) { in.ElementMult = -20 }, MinDamage},
		{"防御が上回っても最低保証", func(in *DamageInput) { in.Defense = 100 }, MinDamage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := base
			tt.edit(&in)
			assert.Equal(t, tt.want, Damage(in))
		})
	}
}
