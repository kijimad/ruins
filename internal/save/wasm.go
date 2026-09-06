//go:build js && wasm

package save

import "fmt"

// WASM は体験版でセーブ・ロードを持たない。ゲーム本体はこのビルドで save パッケージを参照しないが、
// go build ./... はパッケージ単体をビルドするので、プラットフォーム別メソッドの実体をここに置く。
// いずれも永続化せず、呼ばれたらエラーか空を返す。localStorage への保存機構は載せない。

// errNoPersistence は WASM 体験版で永続化が無いことを示す
var errNoPersistence = fmt.Errorf("save is not available in the WASM demo build")

// initImpl は WASM では何もしない
func (sm *SerializationManager) initImpl() error {
	return nil
}

// saveDataImpl は WASM では保存しない
func (sm *SerializationManager) saveDataImpl(_ string, _ []byte) error {
	return errNoPersistence
}

// loadDataImpl は WASM では読み込まない
func (sm *SerializationManager) loadDataImpl(_ string) ([]byte, error) {
	return nil, errNoPersistence
}

// saveFileExistsImpl は WASM では常にセーブ無しを返す
func (sm *SerializationManager) saveFileExistsImpl(_ string) bool {
	return false
}

// listSavesImpl は WASM では空一覧を返す
func (sm *SerializationManager) listSavesImpl() ([]string, error) {
	return nil, nil
}

// deleteSaveImpl は WASM では何もしない
func (sm *SerializationManager) deleteSaveImpl(_ string) error {
	return errNoPersistence
}
