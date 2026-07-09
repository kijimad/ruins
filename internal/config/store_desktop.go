//go:build !js || !wasm

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// userConfigDirName はユーザー設定を格納するサブディレクトリ名
const userConfigDirName = "ruins"

// userConfigFileName はユーザー設定ファイル名
const userConfigFileName = "settings.toml"

// userConfigPath は永続化するユーザー設定ファイルの絶対パスを返す。
// OS標準の設定ディレクトリ配下にアプリ専用のサブディレクトリを切ってその中に置く。
// Linuxでは ~/.config/ruins/settings.toml となる。
func userConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("設定ディレクトリの取得に失敗しました: %w", err)
	}
	return filepath.Join(dir, userConfigDirName, userConfigFileName), nil
}

// readSettings は設定ファイルの内容を返す。ファイルが無ければ ok=false を返す。
func readSettings() ([]byte, bool, error) {
	path, err := userConfigPath()
	if err != nil {
		return nil, false, err
	}
	return readSettingsFrom(path)
}

// readSettingsFrom は指定パスから設定を読み込む。パスを引数に取ることでテストできる。
func readSettingsFrom(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}
	return data, true, nil
}

// writeSettings は設定ファイルへ書き込む。ディレクトリが無ければ作成する。
func writeSettings(data []byte) error {
	path, err := userConfigPath()
	if err != nil {
		return err
	}
	return writeSettingsTo(path, data)
}

// writeSettingsTo は指定パスへ書き込む。一時ファイルへ書いてから rename することで、
// 書き込み途中のクラッシュによる破損を防ぐ。パスを引数に取ることでテストできる。
func writeSettingsTo(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("設定ディレクトリの作成に失敗しました: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("設定ファイルの書き込みに失敗しました: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("設定ファイルの置き換えに失敗しました: %w", err)
	}
	return nil
}

// settingsExist は設定ファイルが存在するかを返す。
func settingsExist() (bool, error) {
	path, err := userConfigPath()
	if err != nil {
		return false, err
	}
	return settingsExistAt(path)
}

// settingsExistAt は指定パスにファイルが存在するかを返す。パスを引数に取ることでテストできる。
func settingsExistAt(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("設定ファイルの存在確認に失敗しました: %w", err)
}
