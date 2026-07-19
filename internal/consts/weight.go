package consts

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MilligramPerKg は 1kg あたりのミリグラム数
const MilligramPerKg = 1_000_000

// MilligramPerGram は 1g あたりのミリグラム数
const MilligramPerGram = 1_000

// Milligram は重量のミリグラム単位。整数で正確に扱い、表示は String/KgString に集約する。
// float の丸め誤差を避けるため所持重量の加減算・比較はこの整数で行う。
// 重量は非負を前提とする。ParseWeight は負値を弾き、通常の計算経路でも負にはならない。
type Milligram int

// MilligramFromKg は kg を Milligram に変換する。丸めて整数化する
func MilligramFromKg(kg float64) Milligram {
	return Milligram(math.Round(kg * MilligramPerKg))
}

// String は値の大きさに応じた最適な単位（kg/g/mg）で表示する。
// 単一アイテムの重量など幅の広い値に使う。1kg以上はkg、1g以上はg、それ未満はmg。
// 数値は FormatFloat の 'f' 最短表記で末尾ゼロを落とす。指数表記にはしない
func (m Milligram) String() string {
	switch {
	case m >= MilligramPerKg:
		return strconv.FormatFloat(float64(m)/MilligramPerKg, 'f', -1, 64) + IconKg
	case m >= MilligramPerGram:
		return strconv.FormatFloat(float64(m)/MilligramPerGram, 'f', -1, 64) + IconG
	default:
		return fmt.Sprintf("%d%s", int(m), IconMg)
	}
}

// KgString は常に kg 表記で表示する。合計所持重量など単位を揃えたい箇所に使う。
// float への変換はこの表示処理の内部だけに閉じ、演算は Milligram の int で行う
func (m Milligram) KgString() string {
	return fmt.Sprintf("%.2f%s", float64(m)/MilligramPerKg, IconKg)
}

// weightUnits は単位文字列を Milligram への係数へ対応させる。
// mg は係数1なので、"0.5 mg" のような小数入力は丸められる。データは整数mgを前提とする
var weightUnits = map[string]float64{
	"mg": 1,
	"g":  MilligramPerGram,
	"kg": MilligramPerKg,
}

// ParseWeight は "500 g" "2 kg" "1 mg" のような単位付き文字列を Milligram に変換する。
// 数値と単位は空白で区切る。単位は mg/g/kg のいずれか。負値は許さない
func ParseWeight(s string) (Milligram, error) {
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, fmt.Errorf("重量の形式が不正です: %q（例: \"500 g\"）", s)
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("重量の数値が不正です: %q", s)
	}
	if value < 0 {
		return 0, fmt.Errorf("重量は負にできません: %q", s)
	}
	factor, ok := weightUnits[fields[1]]
	if !ok {
		return 0, fmt.Errorf("重量の単位が不正です: %q（mg/g/kg のいずれか）", s)
	}
	return Milligram(math.Round(value * factor)), nil
}
