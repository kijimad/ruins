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

const tomlExt = ".toml"

// LayoutStore はレイアウト・チャンクファイルの管理を担当する
// assets.FSを経由せず直接ファイルシステムからTOMLを読み書きする
type LayoutStore struct {
	mu   sync.RWMutex
	dirs []string // レイアウト/チャンク/施設ディレクトリのパス群
}

// NewLayoutStore はLayoutStoreを生成する
func NewLayoutStore(dirs []string) (*LayoutStore, error) {
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("ディレクトリが存在しません: %s: %w", dir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("ディレクトリではありません: %s", dir)
		}
	}
	return &LayoutStore{dirs: dirs}, nil
}

// LayoutFileEntry はファイル内のチャンク情報
type LayoutFileEntry struct {
	Dir      string
	FileName string
	Chunks   []maptemplate.ChunkTemplate
}

// List はすべてのチャンクテンプレートをファイル単位で返す
func (ls *LayoutStore) List() ([]LayoutFileEntry, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	var entries []LayoutFileEntry
	for _, dir := range ls.dirs {
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range dirEntries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != tomlExt {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			chunks, err := ls.loadFile(path)
			if err != nil {
				continue
			}
			entries = append(entries, LayoutFileEntry{
				Dir:      filepath.Base(dir),
				FileName: entry.Name(),
				Chunks:   chunks,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir < entries[j].Dir
		}
		return entries[i].FileName < entries[j].FileName
	})

	return entries, nil
}

// GetChunk はディレクトリ名+ファイル名+チャンク名で特定のチャンクを取得する
func (ls *LayoutStore) GetChunk(dirName, fileName, chunkName string) (*maptemplate.ChunkTemplate, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	path, err := ls.resolvePath(dirName, fileName)
	if err != nil {
		return nil, err
	}

	chunks, err := ls.loadFile(path)
	if err != nil {
		return nil, err
	}

	for i := range chunks {
		if chunks[i].Name == chunkName {
			return &chunks[i], nil
		}
	}
	return nil, fmt.Errorf("チャンク '%s' が見つかりません", chunkName)
}

// SaveChunk はチャンクのmap内容を更新して保存する。
// validateがnilでなければ、保存前にチャンクを検証する
func (ls *LayoutStore) SaveChunk(dirName, fileName, chunkName, mapContent string, validate func(*maptemplate.ChunkTemplate) error) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	path, err := ls.resolvePath(dirName, fileName)
	if err != nil {
		return err
	}

	chunks, err := ls.loadFile(path)
	if err != nil {
		return err
	}

	found := false
	for i := range chunks {
		if chunks[i].Name == chunkName {
			if validate != nil {
				if err := validate(&chunks[i]); err != nil {
					return err
				}
			}
			chunks[i].Map = mapContent
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("チャンク '%s' が見つかりません", chunkName)
	}

	file := maptemplate.ChunkTemplateFile{Chunks: chunks}
	data, err := maptemplate.MarshalChunkTemplateFile(file)
	if err != nil {
		return fmt.Errorf("TOMLマーシャルエラー: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// BuildTemplateLoader はファイルシステムからTemplateLoaderを構築する
// プレビュー生成に使う
func (ls *LayoutStore) BuildTemplateLoader(paletteDir string) (*maptemplate.TemplateLoader, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	loader := maptemplate.NewTemplateLoader()

	// チャンクを登録する
	for _, dir := range ls.dirs {
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range dirEntries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != tomlExt {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			f, err := os.Open(path)
			if err != nil {
				continue
			}
			chunks, err := loader.Load(f)
			_ = f.Close()
			if err != nil {
				continue
			}
			for i := range chunks {
				loader.RegisterChunk(&chunks[i])
			}
		}
	}

	// パレットを登録する
	palEntries, err := os.ReadDir(paletteDir)
	if err != nil {
		return nil, fmt.Errorf("パレットディレクトリ読み込みエラー: %w", err)
	}
	palLoader := maptemplate.NewPaletteLoader()
	for _, entry := range palEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != tomlExt {
			continue
		}
		path := filepath.Join(paletteDir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		p, err := palLoader.Load(f)
		_ = f.Close()
		if err != nil {
			continue
		}
		loader.RegisterPalette(p)
	}

	return loader, nil
}

func (ls *LayoutStore) loadFile(path string) ([]maptemplate.ChunkTemplate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ファイル読み込みエラー: %w", err)
	}
	defer func() { _ = f.Close() }()

	loader := maptemplate.NewTemplateLoader()
	return loader.Load(f)
}

func (ls *LayoutStore) resolvePath(dirName, fileName string) (string, error) {
	for _, dir := range ls.dirs {
		if filepath.Base(dir) == dirName {
			path := filepath.Join(dir, fileName)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("ファイルが見つかりません: %s/%s", dirName, fileName)
}

// FileKey はファイル特定用のキー文字列を生成する
func FileKey(dirName, fileName string) string {
	return dirName + "/" + strings.TrimSuffix(fileName, tomlExt)
}
