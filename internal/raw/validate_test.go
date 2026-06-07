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

	err = ValidateRaws(master)
	assert.NoError(t, err)
}

func TestValidateRaws_ValidItem(t *testing.T) {
	t.Parallel()

	raws := makeItemRaws(func(*oapi.Item) {})
	err := ValidateRaws(raws)
	assert.NoError(t, err)
}

func TestValidateRaws_EmptyRaws(t *testing.T) {
	t.Parallel()

	err := ValidateRaws(oapi.Raws{})
	assert.NoError(t, err)
}

func TestValidateRaws_InvalidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raws    oapi.Raws
		wantErr string
	}{
		{
			name: "名前が長すぎる",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Name = "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん12345678901234567890"
			}),
			wantErr: "maximum string length is 50",
		},
		{
			name: "命中率が範囲外",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.Accuracy = 999
			}),
			wantErr: "number must be at most 100",
		},
		{
			name: "スプライトキーのパターン不正",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.SpriteKey = "INVALID-KEY!"
			}),
			wantErr: `string doesn't match the regular expression`,
		},
		{
			name: "不正な攻撃種別",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.AttackCategory = "INVALID"
			}),
			wantErr: "value is not one of the allowed values",
		},
		{
			name: "不正な属性",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.Element = "INVALID_ELEMENT"
			}),
			wantErr: "value is not one of the allowed values",
		},
		{
			name: "ダメージが負の値",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.Damage = -1
			}),
			wantErr: "number must be at least 0",
		},
		{
			name: "攻撃回数がゼロ",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.AttackCount = 0
			}),
			wantErr: "number must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRaws(tt.raws)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "バリデーションエラー")
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// makeItemRaws は正常なアイテムを1つ持つRawsを生成し、modifyで値を改変する
func makeItemRaws(modify func(*oapi.Item)) oapi.Raws {
	item := oapi.Item{
		Name:            "テスト武器",
		Description:     "テスト用の武器",
		SpriteSheetName: "test_sheet",
		SpriteKey:       "test_key",
		Value:           100,
		Melee: &oapi.Melee{
			Accuracy:       80,
			Damage:         10,
			AttackCount:    1,
			Element:        "NONE",
			AttackCategory: "SWORD",
			Cost:           5,
			TargetGroup:    "ENEMY",
			TargetNum:      "SINGLE",
		},
	}
	modify(&item)
	items := []oapi.Item{item}
	return oapi.Raws{Items: &items}
}
