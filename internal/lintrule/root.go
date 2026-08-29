package lintrule

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrModuleRootNotFound は go.mod を上位へ辿っても見つからないことを表す。
var ErrModuleRootNotFound = errors.New("go.mod が見つからない")

// ModuleRoot は go.mod のあるディレクトリを現在位置から上位へ辿って返す。
// 検査はリポジトリ全体を歩くので、テストの実行位置に依存しない起点が要る。
func ModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrModuleRootNotFound
		}
		dir = parent
	}
}
