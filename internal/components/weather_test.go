package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// allWeatherKinds は全天候種。種を足したらここにも足す。網羅テストの単一の真実にする。
var allWeatherKinds = []WeatherKind{
	WeatherClear, WeatherCloudy, WeatherRain, WeatherSnow, WeatherBlizzard, WeatherColdSnap,
}

func TestWeatherKind_全種が効果と名前と厳しさを持つ(t *testing.T) {
	t.Parallel()
	assert.Len(t, allWeatherKinds, NumWeatherKind, "allWeatherKinds は全種を列挙する")
	for _, k := range allWeatherKinds {
		assert.NotEmpty(t, k.String(), "%d は名前を持つ", k)
		assert.GreaterOrEqual(t, k.Severity(), 0, "%s の厳しさは非負", k)
	}
}

func TestWeatherKind_厳しさは寒さの段階に沿う(t *testing.T) {
	t.Parallel()
	// 雨は降水だが寒さ軸では軽いので0。並び順とは一致しない
	assert.Equal(t, 0, WeatherClear.Severity())
	assert.Equal(t, 0, WeatherCloudy.Severity())
	assert.Equal(t, 0, WeatherRain.Severity())
	assert.Equal(t, 1, WeatherSnow.Severity())
	assert.Equal(t, 2, WeatherBlizzard.Severity())
	assert.Equal(t, 2, WeatherColdSnap.Severity())
}

func TestWeatherKind_荒天ほど寒く視程が狭い(t *testing.T) {
	t.Parallel()
	// 晴れは暖かく、吹雪は寒く視程が大きく落ちる。寒波は酷寒だが視界は保たれる
	assert.Positive(t, WeatherClear.Effect().TempModifier, "晴れは暖かい")
	assert.Negative(t, WeatherBlizzard.Effect().TempModifier, "吹雪は寒い")
	assert.Less(t, WeatherColdSnap.Effect().TempModifier, WeatherBlizzard.Effect().TempModifier, "寒波は吹雪より酷寒")
	assert.Negative(t, WeatherBlizzard.Effect().VisionDelta, "吹雪は視程を削る")
	assert.Zero(t, WeatherColdSnap.Effect().VisionDelta, "寒波は視界を保つ")
}

func TestWeatherKind_String_未知はpanicする(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { _ = WeatherKind(999).String() }, "未知の種類は panic して漏れを露見させる")
}
