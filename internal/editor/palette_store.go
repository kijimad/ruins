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

// PaletteStore はパレットファイルのディレクトリを管理する。
// 起動時に全ファイルをメモリに読み込み、変更時にメモリとファイルの両方を更新する
type PaletteStore struct {
	mu       sync.RWMutex
	dir      string
	palettes []maptemplate.Palette // メモリに保持するパレットデータ
}

// NewPaletteStore はパレットストアを生成する。全ファイルをメモリに読み込む
func NewPaletteStore(dir string) (*PaletteStore, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("パレットディレクトリが存在しません: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("パレットパスがディレクトリではありません: %s", dir)
	}

	ps := &PaletteStore{dir: dir}
	if err := ps.loadAll(); err != nil {
		return nil, err
	}
	return ps, nil
}

// Dir はパレットディレクトリのパスを返す
func (ps *PaletteStore) Dir() string {
	return ps.dir
}

// loadAll は全パレットファイルを読み込んでメモリに保持する
func (ps *PaletteStore) loadAll() error {
	entries, err := os.ReadDir(ps.dir)
	if err != nil {
		return fmt.Errorf("パレットディレクトリ読み込みエラー: %w", err)
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

	ps.sortPalettes(palettes)
	ps.palettes = palettes
	return nil
}

func (ps *PaletteStore) sortPalettes(palettes []maptemplate.Palette) {
	sort.Slice(palettes, func(i, j int) bool {
		return palettes[i].ID < palettes[j].ID
	})
}

// List はすべてのパレットをID順で返す
func (ps *PaletteStore) List() ([]maptemplate.Palette, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.palettes, nil
}

// Get はIDでパレットを取得する
func (ps *PaletteStore) Get(id string) (*maptemplate.Palette, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for i := range ps.palettes {
		if ps.palettes[i].ID == id {
			return &ps.palettes[i], nil
		}
	}
	return nil, fmt.Errorf("パレットが見つかりません: %s", id)
}

// safePath はIDからパストラバーサルを防いだファイルパスを返す
func (ps *PaletteStore) safePath(id string) (string, error) {
	safe := filepath.Base(id)
	if safe != id || safe == "." || safe == ".." || strings.ContainsAny(safe, `/\`) {
		return "", fmt.Errorf("不正なパレットIDです: %q", id)
	}
	absDir, err := filepath.Abs(ps.dir)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(absDir, safe+".toml")
	if !strings.HasPrefix(joined, absDir+string(filepath.Separator)) {
		return "", fmt.Errorf("不正なパレットIDです: %q", id)
	}
	return joined, nil
}

// Save はパレットをメモリとファイルの両方に保存する
func (ps *PaletteStore) Save(p *maptemplate.Palette) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// ファイルに書き出す
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

	// メモリを更新する
	found := false
	for i := range ps.palettes {
		if ps.palettes[i].ID == p.ID {
			ps.palettes[i] = *p
			found = true
			break
		}
	}
	if !found {
		ps.palettes = append(ps.palettes, *p)
		ps.sortPalettes(ps.palettes)
	}

	return nil
}

// Delete はパレットをメモリとファイルの両方から削除する
func (ps *PaletteStore) Delete(id string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// ファイルを削除する
	path, err := ps.safePath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("パレットファイル削除エラー: %w", err)
	}

	// メモリから削除する
	for i := range ps.palettes {
		if ps.palettes[i].ID == id {
			ps.palettes = append(ps.palettes[:i], ps.palettes[i+1:]...)
			break
		}
	}

	return nil
}
