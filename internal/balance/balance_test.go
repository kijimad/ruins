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

func TestLoadWeaponFromItem(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("近接武器を読み込める", func(t *testing.T) {
		t.Parallel()
		ws, err := LoadWeaponFromItem(master, "bare_hands")
		require.NoError(t, err)
		assert.False(t, ws.IsRanged)
	})

	t.Run("存在しないアイテムはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadWeaponFromItem(master, "存在しない武器")
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
		assert.Equal(t, raw.KeyNotFoundError{Key: "存在しない武器", Collection: "Items"}, notFound)
	})
}

func TestLoadEnemyWeapon(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("スライムの武器を読み込める", func(t *testing.T) {
		t.Parallel()
		ws, err := LoadEnemyWeapon(master, "slime")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, ws.Damage, 0)
	})

	t.Run("存在しない敵はエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadEnemyWeapon(master, "存在しない敵")
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

func TestMean(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 3.0, Mean([]int{1, 2, 3, 4, 5}), 0.01)
	assert.InDelta(t, 0.0, Mean(nil), 0.01)
}
