// Package designdoc は docs/design 下の設計ドキュメントの frontmatter を扱う。
//
// # 責務
//   - 設計ドキュメント冒頭の YAML frontmatter の解析と検証
//   - 進捗チェックボックスからの status の決定的な導出
//   - frontmatter を欠くドキュメントへの決定的な付与、すなわち backfill
//
// # 仕様
//   - frontmatter は status・tags・auto の3フィールドを持つ。status は Issue の open/closed に相当し、
//     auto はルーチンが無人で着手してよいかを表す。
//   - status の導出は `## 進捗` のチェックボックスだけを根拠にする。LLM 推論を挟まないため、
//     同じ入力に対して常に同じ結果を返す。これにより一括付与でブレが出ない。
//
// # 設計意図
//
// GitHub Issue を運用せず docs/design をバックログ兼 Issue トラッカーとして使うための土台。
// 構造は本パッケージが決定的に守り、意味付け、すなわち tags と auto の選択だけを人の判断に残す。
package designdoc
