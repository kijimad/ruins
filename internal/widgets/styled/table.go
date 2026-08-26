package styled

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// TextAlign は一覧セル内のテキスト揃え方向を表す
type TextAlign int

// テキスト揃え方向の定数
const (
	AlignLeft TextAlign = iota
	AlignRight
)

// Cell は一覧の1セル。Icon が非 nil ならアイコンセル、nil なら文字列セル。
// アイコンは任意の列に置けるので、行はアイコンと文字列が混ざった一様な []Cell になる。
// 将来アイコンと文字列を1セルへ併記したくなったら両方を持たせて拡張できる
type Cell struct {
	Text string
	Icon *ebiten.Image
}

// TextCell は文字列を表すセルを返す
func TextCell(s string) Cell { return Cell{Text: s} }

// IconCell はアイコンを表すセルを返す。img が nil のときは透明セルになり、桁だけ合う
func IconCell(img *ebiten.Image) Cell { return Cell{Icon: img} }

// TextCells は文字列だけの行を手早く組む
func TextCells(ss ...string) []Cell {
	cells := make([]Cell, len(ss))
	for i, s := range ss {
		cells[i] = TextCell(s)
	}
	return cells
}
