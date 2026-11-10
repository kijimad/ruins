package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// allWeatherKinds は全天候種。種を足したらここにも足す。網羅テストの単一の真実にする。
var allWeatherKinds = []WeatherKind{
	WeatherClear, WeatherCloudy, WeatherRain, WeatherSnow, WeatherBlizzard, WeatherColdSnap,
}

func TestWeatherKind_全種が名前を持つ(t *testing.T) {
	t.Parallel()
	assert.Len(t, allWeatherKinds, NumWeatherKind, "allWeatherKinds は全種を列挙する")
	for _, k := range allWeatherKinds {
		assert.NotEmpty(t, k.String(), "%d は名前を持つ", k)
	}
}

func TestWeatherKind_String_未知はpanicする(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { _ = WeatherKind(999).String() }, "未知の種類は panic して漏れを露見させる")
}
