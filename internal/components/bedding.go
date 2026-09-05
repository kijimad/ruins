package components

import "github.com/kijimaD/ruins/internal/consts"

// Bedding は寝具の性能を持つ。bed などの prop が持ち、睡眠効率を決める。
// 眠るとき足元か隣接の Bedding から Quality を読み、無ければ地べたの100として扱う
type Bedding struct {
	Quality consts.Percent // 睡眠効率。100が基準で、高いほど短く深く眠れる
}
