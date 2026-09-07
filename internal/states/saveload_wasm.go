//go:build js && wasm

package states

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// WASM は永続化を持てないので、セーブ・ロードを常に出さない。ok を偽で返してメインメニューの
// ロード項目とダンジョンメニューのセーブ項目を落とす。NewSaveMenuState・NewLoadMenuState と
// save パッケージへの参照はこのビルドには含めず、機構ごとバイナリから外す。体験版フラグには依らない。

func loadMainMenuItem(_ w.World) (mainMenuItem, bool) {
	return mainMenuItem{}, false
}

func dungeonSaveMenuChoice(_ w.World) (Choice, bool) {
	return Choice{}, false
}
