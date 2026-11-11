package components

import (
	"math"

	"github.com/kijimaD/ruins/internal/i18n"
)

// TimeOfDayDef は時間帯1つの定義。raw の timesOfDay の1行から組み立てる
type TimeOfDayDef struct {
	Name         i18n.Msgid // 表示名
	TempModifier int        // 世界温度への加算。℃
	Daylight     float64    // 時間帯の中心での地上の日照の明るさ
	AmbientColor [3]float64 // 時間帯の中心での地上の環境光の色。明るさは Daylight が持つ
}

// SeasonDef は季節1つの定義。raw の seasons の1行から組み立てる
type SeasonDef struct {
	Name i18n.Msgid // 表示名
	Temp int        // 世界温度。℃。春と秋は年の前半と後半の中点、夏は頂、冬は底
}

// CalendarDefs は時間帯と季節ごとの表示名、気温、明るさの表。読み込み時に raw から1度だけ組み立て、以後は変えない
type CalendarDefs struct {
	TimesOfDay [NumTimeOfDay]TimeOfDayDef
	Seasons    [NumSeason]SeasonDef
}

// SeasonalTemperatureForDay は経過日 day の季節による世界温度のベース値を返す。
// 春開始の区分正弦波で、春と秋が肩、夏がピーク、冬が底になる。四半期ごとに肩とピークを
// 結ぶため肩の高さを春と秋で変えられ、振幅も暖側と寒側で非対称になる。季節は保存せず日数から導く。
// 緯度勾配・時間帯・遮蔽を含まない季節変動そのもので、バランス導出が年間の寒暖の振れを見るのにも使う。
func (d CalendarDefs) SeasonalTemperatureForDay(day int) int {
	// day 1 を春の中点かつ上昇位相の起点にする。位相の比較のため1年周期へ折り返す
	phase := math.Mod(2*math.Pi*float64(day-1)/daysPerYear, 2*math.Pi)
	s := math.Sin(phase)

	// 年の前半の肩は春、夏ピークから冬底を挟む後半の肩は秋
	mid := float64(d.Seasons[SeasonSpring].Temp)
	if phase >= math.Pi/2 && phase < 3*math.Pi/2 {
		mid = float64(d.Seasons[SeasonAutumn].Temp)
	}

	if s >= 0 {
		return int(math.Round(mid + (float64(d.Seasons[SeasonSummer].Temp)-mid)*s))
	}
	return int(math.Round(mid + (mid-float64(d.Seasons[SeasonWinter].Temp))*s))
}
