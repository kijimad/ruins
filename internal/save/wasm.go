//go:build js && wasm

package save

import (
	"encoding/base64"
	"fmt"
	"strings"
	"syscall/js"
)

// initImpl はWASM環境での初期化処理
func (sm *SerializationManager) initImpl() error {
	return nil
}

// saveDataImpl はWASM環境でローカルストレージにデータを保存する。
// localStorage は文字列しか持てないので、gzip バイト列を base64 で包んで書き込む。
func (sm *SerializationManager) saveDataImpl(slotName string, data []byte) error {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return fmt.Errorf("localStorage is not available")
	}

	key := fmt.Sprintf("ruins-savedata-%s", slotName)
	localStorage.Call("setItem", key, base64.StdEncoding.EncodeToString(data))

	return nil
}

// loadDataImpl はWASM環境でローカルストレージからデータを読み込む
func (sm *SerializationManager) loadDataImpl(slotName string) ([]byte, error) {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return nil, fmt.Errorf("localStorage is not available")
	}

	key := fmt.Sprintf("ruins-savedata-%s", slotName)
	item := localStorage.Call("getItem", key)
	if item.IsNull() {
		return nil, fmt.Errorf("save data not found for slot: %s", slotName)
	}

	// base64 を解いて gzip バイト列へ戻す。圧縮解除は共通層が行う
	raw, err := base64.StdEncoding.DecodeString(item.String())
	if err != nil {
		return nil, fmt.Errorf("failed to decode save data: %w", err)
	}
	return raw, nil
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
