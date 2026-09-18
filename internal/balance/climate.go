package balance

import gc "github.com/kijimaD/ruins/internal/components"

// WorldTemperatureAtDay は経過日 day の屋外世界温度のベース値を返す。components.SeasonalTemperatureForDay を
// 単一出典で参照する季節変動そのもので、生存の寒さ圧の上流入力になる。
func WorldTemperatureAtDay(day int) int {
	return gc.SeasonalTemperatureForDay(day)
}
