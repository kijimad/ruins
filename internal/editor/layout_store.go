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

// LayoutStore はレイアウト・チャンクファイルの管理を担当する。
// 起動時に全ファイルをメモリに読み込み、変更時にメモリとファイルの両方を更新する
type LayoutStore struct {
	mu      sync.RWMutex
	dirs    []string          // レイアウト/チャンク/施設ディレクトリのパス群
	entries []LayoutFileEntry // メモリに保持するチャンクデータ
	paths   map[string]string // "dir/file.toml" → 絶対パスの逆引き
}

// LayoutFileEntry はファイル内のチャンク情報
type LayoutFileEntry struct {
	Dir      string
	FileName string
	Chunks   []maptemplate.ChunkTemplate
}

// NewLayoutStore はLayoutStoreを生成する。全ファイルをメモリに読み込む
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
	ls := &LayoutStore{
		dirs:  dirs,
		paths: make(map[string]string),
	}
	if err := ls.loadAll(); err != nil {
		return nil, err
	}
	return ls, nil
}

// loadAll は全ディレクトリからファイルを読み込んでメモリに保持する
func (ls *LayoutStore) loadAll() error {
	var entries []LayoutFileEntry
	for _, dir := range ls.dirs {
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("ディレクトリ読み込みエラー: %s: %w", dir, err)
		}
		dirBase := filepath.Base(dir)
		for _, entry := range dirEntries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != tomlExt {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			chunks, err := loadChunkFile(path)
			if err != nil {
				continue
			}
			entries = append(entries, LayoutFileEntry{
				Dir:      dirBase,
				FileName: entry.Name(),
				Chunks:   chunks,
			})
			ls.paths[dirBase+"/"+entry.Name()] = path
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir < entries[j].Dir
		}
		return entries[i].FileName < entries[j].FileName
	})

	ls.entries = entries
	return nil
}

// List はすべてのチャンクテンプレートをファイル単位で返す
func (ls *LayoutStore) List() ([]LayoutFileEntry, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	return ls.entries, nil
}

// GetChunk はディレクトリ名+ファイル名+チャンク名で特定のチャンクを取得する
func (ls *LayoutStore) GetChunk(dirName, fileName, chunkName string) (*maptemplate.ChunkTemplate, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	return ls.findChunk(dirName, fileName, chunkName)
}

// findChunk はロックなしでメモリからチャンクを検索する
func (ls *LayoutStore) findChunk(dirName, fileName, chunkName string) (*maptemplate.ChunkTemplate, error) {
	for i := range ls.entries {
		if ls.entries[i].Dir == dirName && ls.entries[i].FileName == fileName {
			for j := range ls.entries[i].Chunks {
				if ls.entries[i].Chunks[j].Name == chunkName {
					return &ls.entries[i].Chunks[j], nil
				}
			}
			return nil, fmt.Errorf("チャンク '%s' が見つかりません", chunkName)
		}
	}
	return nil, fmt.Errorf("ファイルが見つかりません: %s/%s", dirName, fileName)
}

// SaveChunk はチャンクのmap内容を更新して保存する。
// validateがnilでなければ、保存前にチャンクを検証する
func (ls *LayoutStore) SaveChunk(dirName, fileName, chunkName, mapContent string, validate func(*maptemplate.ChunkTemplate) error) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// メモリ上のチャンクを検索する
	chunk, err := ls.findChunk(dirName, fileName, chunkName)
	if err != nil {
		return err
	}

	if validate != nil {
		if err := validate(chunk); err != nil {
			return err
		}
	}
	chunk.Map = mapContent

	// ファイルに書き出す
	return ls.saveFile(dirName, fileName)
}

// saveFile はメモリ上のエントリをファイルに書き出す
func (ls *LayoutStore) saveFile(dirName, fileName string) error {
	key := dirName + "/" + fileName
	path, ok := ls.paths[key]
	if !ok {
		return fmt.Errorf("ファイルパスが見つかりません: %s", key)
	}

	for i := range ls.entries {
		if ls.entries[i].Dir == dirName && ls.entries[i].FileName == fileName {
			file := maptemplate.ChunkTemplateFile{Chunks: ls.entries[i].Chunks}
			data, err := maptemplate.MarshalChunkTemplateFile(file)
			if err != nil {
				return fmt.Errorf("TOMLマーシャルエラー: %w", err)
			}
			return os.WriteFile(path, data, 0o644)
		}
	}
	return fmt.Errorf("エントリが見つかりません: %s", key)
}

// BuildTemplateLoader はメモリ上のチャンクデータからTemplateLoaderを構築する。
// プレビュー生成に使う
func (ls *LayoutStore) BuildTemplateLoader(paletteDir string) (*maptemplate.TemplateLoader, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	loader := maptemplate.NewTemplateLoader()

	// メモリ上のチャンクを登録する
	for i := range ls.entries {
		for j := range ls.entries[i].Chunks {
			loader.RegisterChunk(&ls.entries[i].Chunks[j])
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

// loadChunkFile はファイルからチャンクテンプレートを読み込む
func loadChunkFile(path string) ([]maptemplate.ChunkTemplate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ファイル読み込みエラー: %w", err)
	}
	defer func() { _ = f.Close() }()

	loader := maptemplate.NewTemplateLoader()
	return loader.Load(f)
}

// FileKey はファイル特定用のキー文字列を生成する
func FileKey(dirName, fileName string) string {
	return dirName + "/" + strings.TrimSuffix(fileName, tomlExt)
}
