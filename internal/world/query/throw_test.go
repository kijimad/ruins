package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestThrowAttackFromWeight(t *testing.T) {
	t.Parallel()

	g := consts.Milligram(consts.MilligramPerKg / 1000)
	tests := []struct {
		name string
		mg   consts.Milligram
		want gc.ThrowAttack
	}{
		{name: "重さが無い物は威力1・命中70", mg: 0, want: gc.ThrowAttack{Damage: 1, Accuracy: 70}},
		{name: "100 g", mg: 100 * g, want: gc.ThrowAttack{Damage: 1, Accuracy: 69}},
		{name: "499 g は丸めて威力0なので最小の1", mg: 499 * g, want: gc.ThrowAttack{Damage: 1, Accuracy: 68}},
		{name: "1.5 kg は丸めて威力2", mg: 1500 * g, want: gc.ThrowAttack{Damage: 2, Accuracy: 62}},
		{name: "3 kg", mg: 3000 * g, want: gc.ThrowAttack{Damage: 3, Accuracy: 55}},
		{name: "5 kg は上限の威力5", mg: 5000 * g, want: gc.ThrowAttack{Damage: 5, Accuracy: 45}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, throwAttackFromWeight(tt.mg))
		})
	}
}
