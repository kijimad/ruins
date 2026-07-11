package raw

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTargetType(t *testing.T) {
	t.Parallel()

	t.Run("空文字列はゼロ値を返す", func(t *testing.T) {
		t.Parallel()
		result := parseTargetType("", "SINGLE")
		assert.Equal(t, gc.TargetType{}, result)
	})

	t.Run("正常な値を変換する", func(t *testing.T) {
		t.Parallel()
		result := parseTargetType("ENEMY", "SINGLE")
		assert.Equal(t, gc.TargetGroupEnemy, result.TargetGroup)
		assert.Equal(t, gc.TargetSingle, result.TargetNum)
	})

	t.Run("ALLYとALLの組み合わせ", func(t *testing.T) {
		t.Parallel()
		result := parseTargetType("ALLY", "ALL")
		assert.Equal(t, gc.TargetGroupAlly, result.TargetGroup)
		assert.Equal(t, gc.TargetAll, result.TargetNum)
	})
}

func TestParseMelee(t *testing.T) {
	t.Parallel()

	t.Run("正常な近接攻撃を変換する", func(t *testing.T) {
		t.Parallel()
		m := &oapi.Melee{
			Accuracy:       80,
			Damage:         25,
			AttackCount:    2,
			Element:        "FIRE",
			AttackCategory: "SWORD",
			Cost:           100,
			TargetGroup:    "ENEMY",
			TargetNum:      "SINGLE",
		}
		result, err := parseMelee(m)
		require.NoError(t, err)
		assert.Equal(t, 80, result.Accuracy)
		assert.Equal(t, 25, result.Damage)
		assert.Equal(t, 2, result.AttackCount)
		assert.Equal(t, gc.ElementTypeFire, result.Element)
		assert.Equal(t, gc.AttackSword, result.AttackCategory)
		assert.Equal(t, 100, result.Cost)
		assert.Equal(t, gc.TargetGroupEnemy, result.TargetType.TargetGroup)
	})

	t.Run("無効な攻撃種別でエラー", func(t *testing.T) {
		t.Parallel()
		m := &oapi.Melee{
			AttackCategory: "INVALID",
		}
		_, err := parseMelee(m)
		assert.Error(t, err)
	})
}

func TestParseFire(t *testing.T) {
	t.Parallel()

	t.Run("正常な射撃攻撃を変換する", func(t *testing.T) {
		t.Parallel()
		tag := oapi.AmmoTag("9mm")
		f := &oapi.Fire{
			Accuracy:       70,
			Damage:         30,
			AttackCount:    1,
			Element:        "THUNDER",
			AttackCategory: "RIFLE",
			Cost:           150,
			TargetGroup:    "ENEMY",
			TargetNum:      "SINGLE",
			MagazineSize:   10,
			ReloadEffort:   50,
			AmmoTag:        &tag,
		}
		result, err := parseFire(f)
		require.NoError(t, err)
		assert.Equal(t, 70, result.Accuracy)
		assert.Equal(t, 30, result.Damage)
		assert.Equal(t, gc.ElementTypeThunder, result.Element)
		assert.Equal(t, gc.AttackRifle, result.AttackCategory)
		assert.Equal(t, 10, result.MagazineSize)
		assert.Equal(t, 50, result.ReloadEffort)
		assert.Equal(t, "9mm", result.AmmoTag)
	})

	t.Run("AmmoTagがnilでも変換できる", func(t *testing.T) {
		t.Parallel()
		f := &oapi.Fire{
			AttackCategory: "HANDGUN",
			MagazineSize:   6,
			ReloadEffort:   30,
		}
		result, err := parseFire(f)
		require.NoError(t, err)
		assert.Empty(t, result.AmmoTag)
	})

	t.Run("無効な攻撃種別でエラー", func(t *testing.T) {
		t.Parallel()
		f := &oapi.Fire{
			AttackCategory: "INVALID",
		}
		_, err := parseFire(f)
		assert.Error(t, err)
	})
}

func TestNewProvidesHealingFromAPI(t *testing.T) {
	t.Parallel()

	t.Run("PERCENTAGE型は倍率で変換する", func(t *testing.T) {
		t.Parallel()
		h := &oapi.ProvidesHealing{
			ValueType: oapi.PERCENTAGE,
			Ratio:     0.5,
		}
		result := newProvidesHealingFromAPI(h)
		assert.Equal(t, gc.HealRatio, result.Kind)
		assert.Equal(t, 0.5, result.Ratio)
	})

	t.Run("デフォルトは絶対量で変換する", func(t *testing.T) {
		t.Parallel()
		h := &oapi.ProvidesHealing{
			Amount: 50,
		}
		result := newProvidesHealingFromAPI(h)
		assert.Equal(t, gc.HealNumeral, result.Kind)
		assert.Equal(t, 50, result.Numeral)
	})
}

func TestNewBookFromAPI(t *testing.T) {
	t.Parallel()

	t.Run("正常な本を変換する", func(t *testing.T) {
		t.Parallel()
		b := &oapi.Book{
			TotalEffort: 100,
			Skill: &oapi.SkillBook{
				TargetSkill:   "sword",
				MaxLevel:      5,
				RequiredLevel: 2,
			},
		}
		result, err := newBookFromAPI(b)
		require.NoError(t, err)
		assert.Equal(t, 100, result.Effort.Max)
		assert.Equal(t, gc.SkillSword, result.Skill.TargetSkill)
		assert.Equal(t, 5, result.Skill.MaxLevel)
		assert.Equal(t, 2, result.Skill.RequiredLevel)
	})

	t.Run("Skillがnilでエラー", func(t *testing.T) {
		t.Parallel()
		b := &oapi.Book{TotalEffort: 100}
		_, err := newBookFromAPI(b)
		assert.Error(t, err)
	})

	t.Run("未定義スキルIDでエラー", func(t *testing.T) {
		t.Parallel()
		b := &oapi.Book{
			TotalEffort: 100,
			Skill: &oapi.SkillBook{
				TargetSkill: "nonexistent",
			},
		}
		_, err := newBookFromAPI(b)
		assert.Error(t, err)
	})
}

func TestToGCLightSource(t *testing.T) {
	t.Parallel()

	t.Run("nilはnilを返す", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, toGCLightSource(nil))
	})

	t.Run("正常な値を変換する", func(t *testing.T) {
		t.Parallel()
		ls := &oapi.LightSource{
			Radius:  5,
			Enabled: true,
			Color:   oapi.RGBAColor{R: 255, G: 128, B: 64, A: 200},
		}
		result := toGCLightSource(ls)
		require.NotNil(t, result)
		assert.Equal(t, 5, int(result.Radius))
		assert.True(t, result.Enabled)
		assert.Equal(t, uint8(255), result.Color.R)
		assert.Equal(t, uint8(128), result.Color.G)
		assert.Equal(t, uint8(64), result.Color.B)
		assert.Equal(t, uint8(200), result.Color.A)
	})
}

func TestKeyNotFoundError(t *testing.T) {
	t.Parallel()

	err := NewKeyNotFoundError("items", "sword")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "items")
	assert.Contains(t, err.Error(), "sword")
}
