package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestMaster(t *testing.T) rawschema.Raws {
	t.Helper()
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "raw.tomlの読み込みに失敗")
	return master
}

func TestLoadCombatantFromMember(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("プレイヤーのステータスを読み込める", func(t *testing.T) {
		t.Parallel()
		stats, err := LoadCombatantFromMember(master, "ash")
		require.NoError(t, err)
		// HP = 30 + 5*8 + 5 + 5 = 80
		assert.Equal(t, 80, stats.HP)
		assert.Equal(t, 5, stats.Strength)
		assert.Equal(t, 5, stats.Agility)
		assert.Equal(t, 3, stats.Defense)
	})

	t.Run("敵のステータスを読み込める", func(t *testing.T) {
		t.Parallel()
		stats, err := LoadCombatantFromMember(master, "slime")
		require.NoError(t, err)
		// HP = 30 + 1*8 + 2 + 2 = 42
		assert.Equal(t, 42, stats.HP)
		assert.Equal(t, 2, stats.Strength)
	})
}

func TestLoadWeaponFromBodyAttack(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("体の攻撃を近接として読み込める", func(t *testing.T) {
		t.Parallel()
		ws, err := LoadWeaponFromBodyAttack(master, raw.BareHands)
		require.NoError(t, err)
		assert.False(t, ws.IsRanged)
		assert.Positive(t, ws.Damage)
	})

	t.Run("存在しない体の攻撃はエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadWeaponFromBodyAttack(master, "存在しない攻撃")
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
		assert.Equal(t, raw.KeyNotFoundError{Key: "存在しない攻撃", Collection: "BodyAttacks"}, notFound)
	})
}

func TestLoadWeaponFromItem(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("近接武器を読み込める", func(t *testing.T) {
		t.Parallel()
		ws, err := LoadWeaponFromItem(master, "iron_sword")
		require.NoError(t, err)
		assert.Equal(t, WeaponStats{Damage: 15, Accuracy: 90}, ws)
	})

	t.Run("存在しないアイテムはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadWeaponFromItem(master, "存在しない武器")
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
		assert.Equal(t, raw.KeyNotFoundError{Key: "存在しない武器", Collection: "Items"}, notFound)
	})
}

func TestLoadWeaponFromMember(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("スライムの武器を読み込める", func(t *testing.T) {
		t.Parallel()
		ws, err := LoadWeaponFromMember(master, "slime")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, ws.Damage, 0)
	})

	t.Run("攻撃表の最も重い行と素手を表ごとに引く", func(t *testing.T) {
		t.Parallel()
		hands := rawschema.Melee{Accuracy: 100, Damage: 3}
		bite := rawschema.Melee{Accuracy: 90, Damage: 11}
		cleaver := rawschema.Melee{Accuracy: 80, Damage: 14}
		raws := rawschema.Raws{
			BodyAttacks: &[]rawschema.BodyAttack{{Id: raw.BareHands, Name: "Bare hands", Melee: hands}, {Id: "bite", Name: "Bite", Melee: bite}},
			Items:       &[]rawschema.Item{{Id: "cleaver", Name: "Cleaver", Melee: &cleaver}},
			CommandTables: &[]rawschema.CommandTable{
				{Id: "biter", Name: "Biter", Entries: []rawschema.CommandTableEntry{{Item: new("cleaver"), Weight: 1}, {BodyAttack: new("bite"), Weight: 2}}},
				{Id: "butcher", Name: "Butcher", Entries: []rawschema.CommandTableEntry{{BodyAttack: new("bite"), Weight: 1}, {Item: new("cleaver"), Weight: 2}}},
			},
			Members: &[]rawschema.Member{
				{Id: "dog", Name: "Dog", CommandTableId: new("biter")},
				{Id: "cook", Name: "Cook", CommandTableId: new("butcher")},
				{Id: "ash", Name: "Ash"},
			},
		}
		tests := []struct {
			member string
			want   WeaponStats
		}{
			{member: "dog", want: WeaponStats{Damage: 11, Accuracy: 90}},
			{member: "cook", want: WeaponStats{Damage: 14, Accuracy: 80}},
			{member: "ash", want: WeaponStats{Damage: 3, Accuracy: 100}},
		}
		for _, tt := range tests {
			ws, err := LoadWeaponFromMember(raws, tt.member)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ws, tt.member)
		}
	})

	t.Run("存在しない敵はエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadWeaponFromMember(master, "存在しない敵")
		require.Error(t, err)
	})
}

func TestLoadCombatantFromMember_Errors(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("存在しないメンバーはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadCombatantFromMember(master, "存在しないメンバー")
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
		assert.Equal(t, raw.KeyNotFoundError{Key: "存在しないメンバー", Collection: "Members"}, notFound)
	})
}
