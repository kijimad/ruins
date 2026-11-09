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

func TestWeatherDefs_Def(t *testing.T) {
	t.Parallel()

	var kinds [NumWeatherKind]WeatherDef
	kinds[WeatherBlizzard] = WeatherDef{Effect: WeatherEffect{TempModifier: -8, VisionDelta: -28}, Severity: 2, SpellTurnsMin: 450, SpellTurnsMax: 900, SeasonWeights: [4]float64{0.3, 0.1, 0.5, 1.2}}
	defs := NewWeatherDefs(kinds)
	assert.Equal(t, kinds[WeatherBlizzard], defs.Def(WeatherBlizzard))
	assert.Equal(t, WeatherDef{}, defs.Def(WeatherClear))
}

func TestWeatherKind_String_未知はpanicする(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { _ = WeatherKind(999).String() }, "未知の種類は panic して漏れを露見させる")
}
