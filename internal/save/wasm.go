//go:build js && wasm

package save

import (
	"fmt"
	"runtime"
	"syscall/js"
)

// initializePlatform はWASM環境での初期化処理（特に何もしない）
func (sm *SerializationManager) initImpl() {
	// WASM環境ではディレクトリ作成は不要
}

// saveDataImpl はWASM環境でローカルストレージにデータを保存する
func (sm *SerializationManager) saveDataImpl(slotName string, data []byte) error {
	if runtime.GOOS != "js" {
		return fmt.Errorf("localStorage is only available in WASM environment")
	}

	// ローカルストレージにアクセス
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return fmt.Errorf("localStorage is not available")
	}

	// キー名を作成（ruins-savedata-{slotName}の形式）
	key := fmt.Sprintf("ruins-savedata-%s", slotName)

	// データを文字列として保存
	localStorage.Call("setItem", key, string(data))

	return nil
}

// loadDataImpl はWASM環境でローカルストレージからデータを読み込む
func (sm *SerializationManager) loadDataImpl(slotName string) ([]byte, error) {
	if runtime.GOOS != "js" {
		return nil, fmt.Errorf("localStorage is only available in WASM environment")
	}

	// ローカルストレージにアクセス
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return nil, fmt.Errorf("localStorage is not available")
	}

	// キー名を作成
	key := fmt.Sprintf("ruins-savedata-%s", slotName)

	// データを取得
	item := localStorage.Call("getItem", key)
	if item.IsNull() {
		return nil, fmt.Errorf("save data not found for slot: %s", slotName)
	}

	return []byte(item.String()), nil
}

// saveFileExistsImpl はWASM環境でセーブファイルが存在するかチェックする
func (sm *SerializationManager) saveFileExistsImpl(slotName string) bool {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return false
	}

	key := fmt.Sprintf("ruins-savedata-%s", slotName)
	item := localStorage.Call("getItem", key)
	return !item.IsNull()
}
