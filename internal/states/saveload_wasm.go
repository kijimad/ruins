//go:build js && wasm

package states

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// WASM は体験版でセーブ・ロードを持たない。ok を偽で返してメインメニューのロード項目と
// ダンジョンメニューのセーブ項目を落とす。NewSaveMenuState・NewLoadMenuState と save パッケージへの
// 参照はこのビルドには含めず、機構ごとバイナリから外す。

func loadMainMenuItem(_ w.World) (mainMenuItem, bool) {
	return mainMenuItem{}, false
}

func dungeonSaveMenuChoice(_ w.World) (Choice, bool) {
	return Choice{}, false
}
