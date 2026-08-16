package consts

import (
	"fmt"
)

// Currency は所持金や価格などの金額。整数で正確に扱い、表示整形は String に閉じる。
// float の丸め誤差を避けるため加減算と比較はこの整数で行う。
// 重量 Milligram と同じ設計で、単位取り違えを型で弾くために素の int と区別する。
type Currency int

// String はカンマ区切りと通貨記号で整形する。3桁ごとにカンマを入れ、負値は先頭に符号を付ける。
func (c Currency) String() string {
	str := fmt.Sprintf("%d", int(c))

	negative := false
	if c < 0 {
		negative = true
		str = str[1:]
	}

	var result string
	for i, ch := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(ch)
	}

	if negative {
		result = "-" + result
	}

	return fmt.Sprintf("%s %s", IconCurrency, result)
}
