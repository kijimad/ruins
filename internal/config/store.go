package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
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

// loadUserConfig は設定ファイルがあれば c.User を上書きする。
// ファイルが存在しない場合はデフォルト値のまま何もしない。読み取り専用で副作用を持たない。
func (c *Config) loadUserConfig() error {
	path, err := userConfigPath()
	if err != nil {
		return err
	}
	return c.loadUserConfigFrom(path)
}

// EnsureUserConfigFile は設定ファイルが存在しない場合にデフォルト値で作成する。
// 初回起動時に設定ファイルを生成してユーザーが編集できるようにする用途で、アプリ起動時に呼ぶ。
func EnsureUserConfigFile() error {
	path, err := userConfigPath()
	if err != nil {
		return err
	}
	return ensureUserConfigFileAt(path)
}

// ensureUserConfigFileAt は指定パスにファイルが無ければデフォルト値で作成する。
// パスを引数に取ることで、実際の設定ディレクトリに依存せずテストできる。
func ensureUserConfigFileAt(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil // 既に存在するので作成しない
	}
	if !errors.Is(err, os.ErrNotExist) {
		// パーミッションエラー等、存在確認自体に失敗した場合は診断できるよう返す
		return fmt.Errorf("設定ファイルの存在確認に失敗しました: %w", err)
	}
	// ファイルが存在しないのでデフォルト値で作成する
	def := &Config{User: DefaultUserConfig()}
	return def.saveUserConfigTo(path)
}

// SaveUserConfig は c.User を設定ファイルへ書き込む。オプション画面での設定変更後に呼ぶ。
func (c *Config) SaveUserConfig() error {
	path, err := userConfigPath()
	if err != nil {
		return err
	}
	return c.saveUserConfigTo(path)
}

// loadUserConfigFrom は指定パスから c.User を上書きする。
// パスを引数に取ることで、実際の設定ディレクトリに依存せずテストできる。
func (c *Config) loadUserConfigFrom(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// c.User を土台に復元するため、ファイルに存在しないフィールドはデフォルト値が残る
	if err := toml.Unmarshal(data, &c.User); err != nil {
		return fmt.Errorf("設定ファイルの解析に失敗しました: %w", err)
	}
	return nil
}

// saveUserConfigTo は c.User を指定パスへ書き込む。ディレクトリが無ければ作成する。
// 一時ファイルへ書いてから rename することで、書き込み途中のクラッシュによる破損を防ぐ。
func (c *Config) saveUserConfigTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("設定ディレクトリの作成に失敗しました: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(c.User); err != nil {
		return fmt.Errorf("設定のエンコードに失敗しました: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("設定ファイルの書き込みに失敗しました: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		// 中途半端な一時ファイルを残さないようベストエフォートで削除する
		_ = os.Remove(tmp)
		return fmt.Errorf("設定ファイルの置き換えに失敗しました: %w", err)
	}
	return nil
}
