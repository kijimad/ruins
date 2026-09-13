package balance

import gc "github.com/kijimaD/ruins/internal/components"

// WorldTemperatureAtDay は経過日 day の屋外世界温度のベース値を返す。緯度勾配・時間帯・遮蔽を
// 含まない季節変動そのもので、components.SeasonalTemperatureForDay を単一出典で参照する。
// 1年 daysPerYear 周期で春から夏ピーク、秋、冬底へ巡る。生存の寒さ圧の上流入力になる。
func WorldTemperatureAtDay(day int) int {
	return gc.SeasonalTemperatureForDay(day)
}
