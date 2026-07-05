//go:build !js || !wasm

package save

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kijimaD/ruins/internal/logger"
)

// initImpl はデスクトップ環境での初期化処理
func (sm *SerializationManager) initImpl() {
	// セーブディレクトリを作成（存在しない場合）
	if err := os.MkdirAll(sm.saveDirectory, 0755); err != nil {
		logger.New(logger.CategorySave).Warn("セーブディレクトリの作成に失敗", "error", err)
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

// listSavesImpl はデスクトップ環境でセーブファイル名の一覧を返す
func (sm *SerializationManager) listSavesImpl() ([]string, error) {
	entries, err := os.ReadDir(sm.saveDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to read save directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if before, ok := strings.CutSuffix(name, ".json"); ok {
			names = append(names, before)
		}
	}
	return names, nil
}

// deleteSaveImpl はデスクトップ環境でセーブファイルを削除する
func (sm *SerializationManager) deleteSaveImpl(slotName string) error {
	fileName := filepath.Join(sm.saveDirectory, slotName+".json")
	if err := os.Remove(fileName); err != nil {
		return fmt.Errorf("failed to delete save file: %w", err)
	}
	return nil
}
