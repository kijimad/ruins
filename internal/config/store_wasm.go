//go:build js && wasm

package config

import (
	"fmt"
	"syscall/js"
)

// settingsStorageKey はローカルストレージ上の設定のキー名
const settingsStorageKey = "ruins-settings"

// readSettings はローカルストレージから設定を読み込む。無ければ ok=false を返す。
func readSettings() ([]byte, bool, error) {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return nil, false, fmt.Errorf("localStorageが利用できません")
	}
	item := localStorage.Call("getItem", settingsStorageKey)
	if item.IsNull() {
		return nil, false, nil
	}
	return []byte(item.String()), true, nil
}

// writeSettings はローカルストレージへ設定を書き込む。
func writeSettings(data []byte) error {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return fmt.Errorf("localStorageが利用できません")
	}
	localStorage.Call("setItem", settingsStorageKey, string(data))
	return nil
}

// settingsExist はローカルストレージに設定が存在するかを返す。
func settingsExist() (bool, error) {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return false, fmt.Errorf("localStorageが利用できません")
	}
	item := localStorage.Call("getItem", settingsStorageKey)
	return !item.IsNull(), nil
}
