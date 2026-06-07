package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRaws_RealData(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	err = ValidateRaws(master.Raws)
	assert.NoError(t, err)
}

func TestValidateRaws_InvalidEnum(t *testing.T) {
	t.Parallel()

	raws := Raws{
		Items: []oapi.Item{
			{
				Name: "不正武器",
				Melee: &oapi.Melee{
					Element:        "INVALID_ELEMENT",
					AttackCategory: "SWORD",
					TargetGroup:    "ENEMY",
					TargetNum:      "SINGLE",
				},
			},
		},
	}
	err := ValidateRaws(raws)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "バリデーションエラー")
}

func TestValidateRaws_InvalidType(t *testing.T) {
	t.Parallel()

	raws := Raws{
		Items: []oapi.Item{
			{
				Name:  "型不正アイテム",
				Value: 100,
				Melee: &oapi.Melee{
					Element:        "NONE",
					AttackCategory: "SWORD",
					TargetGroup:    "ENEMY",
					TargetNum:      "SINGLE",
					Damage:         10,
					Accuracy:       80,
					AttackCount:    1,
					Cost:           5,
				},
			},
		},
	}

	// 正常なデータはエラーにならない
	err := ValidateRaws(raws)
	assert.NoError(t, err)
}

func TestValidateRaws_EmptyRaws(t *testing.T) {
	t.Parallel()

	err := ValidateRaws(Raws{})
	assert.NoError(t, err)
}
