//go:build js && wasm

package save

import "fmt"

// WASM は体験版でセーブ・ロードを持たない。ゲーム本体はこのビルドで save パッケージを参照しないが、
// go build ./... はパッケージ単体をビルドするので、プラットフォーム別メソッドの実体をここに置く。
// いずれも永続化せず、呼ばれたらエラーか空を返す。localStorage への保存機構は載せない。

var errNoPersistence = fmt.Errorf("save is not available in the WASM demo build")

func (sm *SerializationManager) initImpl() error {
	return nil
}

func (sm *SerializationManager) saveDataImpl(_ string, _ []byte) error {
	return errNoPersistence
}

func (sm *SerializationManager) loadDataImpl(_ string) ([]byte, error) {
	return nil, errNoPersistence
}

func (sm *SerializationManager) saveFileExistsImpl(_ string) bool {
	return false
}

func (sm *SerializationManager) listSavesImpl() ([]string, error) {
	return nil, nil
}

func (sm *SerializationManager) deleteSaveImpl(_ string) error {
	return errNoPersistence
}
