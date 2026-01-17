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
	// NumeralType は数値タイプを表す
	NumeralType ValueType = "NUMERAL"
)

// Valid はValueTypeの値が有効かどうかを検証する
func (enum ValueType) Valid() error {
	switch enum {
	case PercentageType, NumeralType:
		return nil
	}
	return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
}
