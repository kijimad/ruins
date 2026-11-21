package raw_test

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectCommandByWeight_重みのある行を選ぶ(t *testing.T) {
	t.Parallel()
	a, b := "A", "B"
	ct := rawschema.CommandTable{
		Name: "TEST",
		Entries: []rawschema.CommandTableEntry{
			{BodyAttack: &a, Weight: 0},
			{Item: &b, Weight: 1},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	got, ok := raw.SelectCommandByWeight(ct, rng)
	require.True(t, ok)
	assert.Equal(t, rawschema.CommandTableEntry{Item: &b, Weight: 1}, got)
}

func TestSelectCommandByWeight_重みが無いときは選ばない(t *testing.T) {
	t.Parallel()
	a := "A"
	ct := rawschema.CommandTable{
		Name:    "TEST",
		Entries: []rawschema.CommandTableEntry{{BodyAttack: &a, Weight: 0}},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	_, ok := raw.SelectCommandByWeight(ct, rng)
	assert.False(t, ok)
}

func TestCommandEntryMelee(t *testing.T) {
	t.Parallel()
	melee := rawschema.Melee{Accuracy: 90, Damage: 11, AttackCount: 1, Cost: 1, AttackCategory: rawschema.AttackCategoryFist, Element: rawschema.ElementNONE, TargetGroup: rawschema.TargetGroupENEMY, TargetNum: rawschema.SINGLE}
	raws := rawschema.Raws{
		BodyAttacks: &[]rawschema.BodyAttack{{Id: "bite", Name: "Bite", Melee: melee}},
		Items: &[]rawschema.Item{
			{Id: "cleaver", Name: "Cleaver", Melee: &melee},
		},
	}
	bite, cleaver := "bite", "cleaver"
	want := &gc.Melee{Accuracy: 90, Damage: 11, AttackCount: 1, Cost: 1, AttackCategory: rawschema.AttackCategoryFist, Element: rawschema.ElementNONE, TargetType: gc.TargetType{TargetGroup: rawschema.TargetGroupENEMY, TargetNum: rawschema.SINGLE}}

	t.Run("体の攻撃を引く", func(t *testing.T) {
		t.Parallel()
		got, name, err := raw.CommandEntryMelee(raws, rawschema.CommandTableEntry{BodyAttack: &bite, Weight: 1})
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, i18n.Msgid("Bite"), name)
	})
	t.Run("アイテムを引く", func(t *testing.T) {
		t.Parallel()
		got, name, err := raw.CommandEntryMelee(raws, rawschema.CommandTableEntry{Item: &cleaver, Weight: 1})
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, i18n.Msgid("Cleaver"), name)
	})
}
