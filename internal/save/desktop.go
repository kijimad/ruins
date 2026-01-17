//go:build !js || !wasm

package save

import (
	"fmt"
	"os"
	"path/filepath"
)

// initImpl はデスクトップ環境での初期化処理
func (sm *SerializationManager) initImpl() {
	// セーブディレクトリを作成（存在しない場合）
	if err := os.MkdirAll(sm.saveDirectory, 0755); err != nil {
		// エラーが発生してもマネージャーは作成する（ログ出力のみ）
		fmt.Printf("Failed to create save directory: %v\n", err)
	}
}

// saveDataImpl はデスクトップ環境でファイルシステムにデータを保存する
func (sm *SerializationManager) saveDataImpl(slotName string, data []byte) error {
	// ファイルに書き込み
	fileName := filepath.Join(sm.saveDirectory, slotName+".json")
	err := os.WriteFile(fileName, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}

	return nil
}

// loadDataImpl はデスクトップ環境でファイルシステムからデータを読み込む
func (sm *SerializationManager) loadDataImpl(slotName string) ([]byte, error) {
	fileName := filepath.Join(sm.saveDirectory, slotName+".json")
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read save file: %w", err)
	}
	return data, nil
}

// saveFileExistsImpl はデスクトップ環境でセーブファイルが存在するかチェックする
func (sm *SerializationManager) saveFileExistsImpl(slotName string) bool {
	fileName := filepath.Join(sm.saveDirectory, slotName+".json")
	_, err := os.Stat(fileName)
	return err == nil
}
