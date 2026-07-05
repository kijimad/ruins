//go:build js && wasm

package save

import (
	"fmt"
	"strings"
	"syscall/js"
)

// initImpl はWASM環境での初期化処理
func (sm *SerializationManager) initImpl() error {
	return nil
}

// saveDataImpl はWASM環境でローカルストレージにデータを保存する
func (sm *SerializationManager) saveDataImpl(slotName string, data []byte) error {
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

// listSavesImpl はWASM環境でセーブデータ名の一覧を返す
func (sm *SerializationManager) listSavesImpl() ([]string, error) {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return nil, fmt.Errorf("localStorage is not available")
	}

	const prefix = "ruins-savedata-"
	length := localStorage.Get("length").Int()
	var names []string
	for i := 0; i < length; i++ {
		key := localStorage.Call("key", i).String()
		if strings.HasPrefix(key, prefix) {
			names = append(names, strings.TrimPrefix(key, prefix))
		}
	}
	return names, nil
}

// deleteSaveImpl はWASM環境でセーブデータを削除する
func (sm *SerializationManager) deleteSaveImpl(slotName string) error {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return fmt.Errorf("localStorage is not available")
	}

	key := fmt.Sprintf("ruins-savedata-%s", slotName)
	localStorage.Call("removeItem", key)
	return nil
}
