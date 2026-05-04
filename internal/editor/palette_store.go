package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/kijimaD/ruins/internal/maptemplate"
)

// PaletteStore はパレットファイルのディレクトリを管理する
type PaletteStore struct {
	mu  sync.RWMutex
	dir string // パレットファイルのディレクトリパス
}

// NewPaletteStore はパレットストアを生成する
func NewPaletteStore(dir string) (*PaletteStore, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("パレットディレクトリが存在しません: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("パレットパスがディレクトリではありません: %s", dir)
	}

	return &PaletteStore{dir: dir}, nil
}

// Dir はパレットディレクトリのパスを返す
func (ps *PaletteStore) Dir() string {
	return ps.dir
}

// List はすべてのパレットをID順で返す
func (ps *PaletteStore) List() ([]maptemplate.Palette, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.listUnsafe()
}

func (ps *PaletteStore) listUnsafe() ([]maptemplate.Palette, error) {
	entries, err := os.ReadDir(ps.dir)
	if err != nil {
		return nil, fmt.Errorf("パレットディレクトリ読み込みエラー: %w", err)
	}

	loader := maptemplate.NewPaletteLoader()
	var palettes []maptemplate.Palette
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		path := filepath.Join(ps.dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		p, err := loader.Load(f)
		_ = f.Close()
		if err != nil {
			continue
		}
		palettes = append(palettes, *p)
	}

	sort.Slice(palettes, func(i, j int) bool {
		return palettes[i].ID < palettes[j].ID
	})

	return palettes, nil
}

// Get はIDでパレットを取得する
func (ps *PaletteStore) Get(id string) (*maptemplate.Palette, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.loadFile(id)
}

// safePath はIDからパストラバーサルを防いだファイルパスを返す
func (ps *PaletteStore) safePath(id string) (string, error) {
	safe := filepath.Base(id)
	if safe != id || safe == "." || safe == ".." || strings.ContainsAny(safe, `/\`) {
		return "", fmt.Errorf("不正なパレットIDです: %q", id)
	}
	return filepath.Join(ps.dir, safe+".toml"), nil
}

func (ps *PaletteStore) loadFile(id string) (*maptemplate.Palette, error) {
	path, err := ps.safePath(id)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("パレットファイルが見つかりません: %w", err)
	}
	defer func() { _ = f.Close() }()

	loader := maptemplate.NewPaletteLoader()
	return loader.Load(f)
}

// Save はパレットをファイルに保存する
func (ps *PaletteStore) Save(p *maptemplate.Palette) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	path, err := ps.safePath(p.ID)
	if err != nil {
		return err
	}

	file := maptemplate.PaletteFile{Palette: *p}
	data, err := maptemplate.MarshalPaletteFile(file)
	if err != nil {
		return fmt.Errorf("パレットTOMLマーシャルエラー: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("パレットファイル書き込みエラー: %w", err)
	}

	return nil
}

// Delete はパレットファイルを削除する
func (ps *PaletteStore) Delete(id string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	path, err := ps.safePath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("パレットファイル削除エラー: %w", err)
	}

	return nil
}
