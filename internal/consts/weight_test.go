package consts_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMilligram_String(t *testing.T) {
	t.Parallel()

	// String は値の大きさに応じて最適な単位を選ぶ
	tests := []struct {
		name string
		mg   consts.Milligram
		want string
	}{
		{"1.5kgはkg", 1_500_000, "1.5" + consts.IconKg},
		{"2kgはkg", 2_000_000, "2" + consts.IconKg},
		{"500gはg", 500_000, "500" + consts.IconG},
		{"50gはg", 50_000, "50" + consts.IconG},
		{"1g未満はmg", 500, "500" + consts.IconMg},
		{"1mg", 1, "1" + consts.IconMg},
		{"ゼロはmg", 0, "0" + consts.IconMg},
		{"大質量でも指数表記にしない", 12_345_678_000_000, "12345678" + consts.IconKg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.mg.String())
		})
	}
}

func TestMilligram_KgString(t *testing.T) {
	t.Parallel()

	// KgString は常に kg 固定で小数2桁
	tests := []struct {
		name string
		mg   consts.Milligram
		want string
	}{
		{"1.5kg", 1_500_000, "1.50" + consts.IconKg},
		{"2kg", 2_000_000, "2.00" + consts.IconKg},
		{"500gも0.50kg", 500_000, "0.50" + consts.IconKg},
		{"ゼロ", 0, "0.00" + consts.IconKg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.mg.KgString())
		})
	}
}

func TestParseWeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want consts.Milligram
	}{
		{"グラム", "500 g", 500_000},
		{"キログラム", "2 kg", 2_000_000},
		{"ミリグラム", "1 mg", 1},
		{"小数キログラム", "3.5 kg", 3_500_000},
		{"1kg未満のグラム", "50 g", 50_000},
		{"小数グラム", "0.5 g", 500},
		{"float誤差を丸める", "0.1 kg", 100_000}, // 0.1*1e6 の丸め誤差を排除
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := consts.ParseWeight(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseWeight_エラー(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"空文字", ""},
		{"単位なし", "500"},
		{"未知の単位", "500 lb"},
		{"数値が不正", "abc g"},
		{"フィールド過多", "1 2 g"},
		{"単位のみ", "kg"},
		{"負値", "-1 kg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := consts.ParseWeight(tt.in)
			assert.Error(t, err)
		})
	}
}

// TestParseWeight_Stringラウンドトリップ は kg 表記でパースと表示が一貫することを確認する
func TestParseWeight_Stringラウンドトリップ(t *testing.T) {
	t.Parallel()

	mg, err := consts.ParseWeight("3.5 kg")
	require.NoError(t, err)
	assert.Equal(t, "3.5"+consts.IconKg, mg.String())    // 最適単位
	assert.Equal(t, "3.50"+consts.IconKg, mg.KgString()) // kg固定
}
