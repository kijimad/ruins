// Package menuframe は画面レベルの UI 足場の寸法を与える。
//
// モーダルの中央配置 CenterWindowRect、ログ領域を避ける基準 logTopY、
// 一覧が1ページに収められる行数 ListCapacity を持つ。
// 実際の描画は states の ViewUI が internal/ui で組む。本パッケージは寸法だけを担う。
package menuframe
