// Package i18n は国際化のマスタデータと訳引きを担う。英語原文を msgid にし、指定言語の訳を返す。
//
// gettext の PO 形式を採り、純 Go の gotext で読む。全言語の訳を Catalog に不変マスタとして構築する。
// Catalog は Resources が保持し、query.T が現在言語とともに引く。日本語は埋め込みの ja.po が持ち、英語は
// 原文そのものなので PO を持たない。未知の言語や未訳は原文の英語へフォールバックする。
//
// 現在言語は本パッケージも Catalog も持たない。ECS シングルトンの UserSettings が保持し、query.T が橋渡しする。
package i18n
