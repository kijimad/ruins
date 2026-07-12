// Package genspec はコンポーネント定義の単一登録表(Registry)を提供する。
//
// # 責務
//
// 全コンポーネント型の登録エントリを、生成先に依存しない純データとして保持する。
// この登録表が source of truth であり、cmd の gencomponents サブコマンドがこれを読んで
// components パッケージの EntitySpec / Components / InitializeComponents / AddEntity を
// 生成する(internal/components/components_gen.go)。
//
// # 使い分け
//
// コンポーネントを追加・変更するときは Registry に1行加えて `make generate` する。
// 型定義そのもの(type X struct{...})は components パッケージに書く。
package genspec
