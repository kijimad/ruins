// Package gorules は golangci-lint の gocritic ruleguard が読み込むカスタムルールを収める。
// このパッケージはアプリから import されず、ruleguard がルール定義として解釈する。
// 通常ビルドにも含まれるが未使用関数となるため、解析対象からは除外している。
package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// adjacentMarkup は同じ Logger への隣接 Markup 呼び出しを禁止する。
// Markup は断片を1つの Logger に積んで自身を返すため、隣接チェーンは1ログ行を
// 断片へ割ったものになり、語順ごとの翻訳を妨げる。1ログ行は1 Markup 文字列にする。
func adjacentMarkup(m dsl.Matcher) {
	m.Import("github.com/kijimaD/ruins/internal/gamelog")
	m.Match(`$x.Markup($a).Markup($b)`).
		Where(m["x"].Type.Is("*gamelog.Logger")).
		Report(`隣接する Markup は1つに統合する。1ログ行は1 Markup 文字列にし、断片連結を避ける`)
}
