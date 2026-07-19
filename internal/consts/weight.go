package consts

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MilligramPerKg は 1kg あたりのミリグラム数
const MilligramPerKg = 1_000_000

// Milligram は重量のミリグラム単位。整数で正確に扱い、String で kg 表記へ統一する。
// float の丸め誤差を避けるため所持重量の加減算・比較はこの整数で行う。
type Milligram int

// MilligramFromKg は kg を Milligram に変換する。丸めて整数化する
func MilligramFromKg(kg float64) Milligram {
	return Milligram(math.Round(kg * MilligramPerKg))
}

// String は kg 表記の文字列を返す。重量表示を一箇所に集約する。
// float への変換はこの表示処理の内部だけに閉じ、演算は Milligram の int で行う
func (m Milligram) String() string {
	return fmt.Sprintf("%.2f%s", float64(m)/MilligramPerKg, IconKg)
}

// weightUnits は単位文字列を Milligram への係数へ対応させる
var weightUnits = map[string]float64{
	"mg": 1,
	"g":  1_000,
	"kg": MilligramPerKg,
}

// ParseWeight は "500 g" "2 kg" "1 mg" のような単位付き文字列を Milligram に変換する。
// 数値と単位は空白で区切る。単位は mg/g/kg のいずれか
func ParseWeight(s string) (Milligram, error) {
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, fmt.Errorf("重量の形式が不正です: %q（例: \"500 g\"）", s)
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("重量の数値が不正です: %q", s)
	}
	factor, ok := weightUnits[fields[1]]
	if !ok {
		return 0, fmt.Errorf("重量の単位が不正です: %q（mg/g/kg のいずれか）", s)
	}
	return Milligram(math.Round(value * factor)), nil
}
