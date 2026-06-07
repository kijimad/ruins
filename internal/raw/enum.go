package raw

import (
	"errors"
	"fmt"
)

// ErrInvalidEnumType はenumに無効な値が指定された場合のエラー
var ErrInvalidEnumType = errors.New("enumに無効な値が指定された")

// ================
// 値タイプ

// ValueType は値のタイプを表す
type ValueType string

const (
	// PercentageType はパーセンテージタイプを表す
	PercentageType ValueType = "PERCENTAGE"
	// AbsoluteType は固定値タイプを表す
	AbsoluteType ValueType = "ABSOLUTE"
	// NumeralType は数値タイプを表す
	NumeralType ValueType = "NUMERAL"
)

// Valid はValueTypeの値が有効かどうかを検証する
func (enum ValueType) Valid() error {
	switch enum {
	case PercentageType, AbsoluteType, NumeralType:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}
