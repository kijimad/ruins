package config

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
)

// ストレージ層（設定の生バイト列の読み書き）はプラットフォームごとに実装が異なる。
// デスクトップはファイル（store_desktop.go）、WASMはローカルストレージ（store_wasm.go）を使う。
// このファイルは TOML の変換と高レベルAPIを提供し、プラットフォームに依存しない。
//
//   - readSettings() ([]byte, bool, error): 保存済みの生データを返す。無ければ ok=false
//   - writeSettings([]byte) error: 生データを永続化する
//   - settingsExist() (bool, error): 保存済みデータが存在するか

// loadUserConfig は永続化された設定を読み込んで c.User を上書きする。
// 保存された設定が無い場合は何もせず、呼び出し前のデフォルト値を維持する。読み取り専用。
func (c *Config) loadUserConfig() error {
	data, ok, err := readSettings()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	// c.User を土台に復元するため、保存に含まれないフィールドはデフォルト値が残る
	if err := toml.Unmarshal(data, &c.User); err != nil {
		return fmt.Errorf("設定の解析に失敗しました: %w", err)
	}
	return nil
}

// SaveUserConfig は c.User を永続化する。オプション画面での設定変更後に呼ぶ。
func (c *Config) SaveUserConfig() error {
	data, err := c.encodeUserConfig()
	if err != nil {
		return err
	}
	return writeSettings(data)
}

// EnsureUserConfigFile は永続化された設定が無い場合にデフォルト値で作成する。
// 初回起動時に設定を生成してユーザーが確認・編集できるようにする用途で、アプリ起動時に呼ぶ。
func EnsureUserConfigFile() error {
	ok, err := settingsExist()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	def := &Config{User: DefaultUserConfig()}
	return def.SaveUserConfig()
}

// encodeUserConfig は c.User を TOML にエンコードする。
func (c *Config) encodeUserConfig() ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(c.User); err != nil {
		return nil, fmt.Errorf("設定のエンコードに失敗しました: %w", err)
	}
	return buf.Bytes(), nil
}
