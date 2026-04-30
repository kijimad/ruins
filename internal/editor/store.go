package editor

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/kijimaD/ruins/internal/raw"
)

// Store はraw.tomlの読み書きを管理する
type Store struct {
	mu   sync.RWMutex
	path string
	raws raw.Raws
}

// NewStore は指定パスのraw.tomlを読み込んでStoreを作成する
func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	bs, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("raw.tomlの読み込みに失敗: %w", err)
	}
	var raws raw.Raws
	if _, err := toml.Decode(string(bs), &raws); err != nil {
		return fmt.Errorf("raw.tomlのパースに失敗: %w", err)
	}
	s.raws = raws
	sortItems(s.raws.Items)
	return nil
}

// sortItems はアイテムを種別の組み合わせ順、同種別内では名前順にソートする
func sortItems(items []raw.Item) {
	sort.SliceStable(items, func(i, j int) bool {
		ki, kj := itemSortKey(items[i]), itemSortKey(items[j])
		if ki != kj {
			return ki < kj
		}
		return items[i].Name < items[j].Name
	})
}

// itemSortKey はアイテムの種別フラグの組み合わせからソートキーを生成する。
// 同じ種別の組み合わせを持つアイテムが隣接して並ぶ
func itemSortKey(item raw.Item) string {
	var key []byte
	flags := []struct {
		present bool
		code    byte
	}{
		{item.Weapon != nil, 'A'},
		{item.Melee != nil, 'B'},
		{item.Fire != nil, 'C'},
		{item.Wearable != nil, 'D'},
		{item.Consumable != nil, 'E'},
		{item.Ammo != nil, 'F'},
		{item.Book != nil, 'G'},
	}
	for _, f := range flags {
		if f.present {
			key = append(key, f.code)
		}
	}
	if len(key) == 0 {
		key = append(key, 'Z')
	}
	return string(key)
}

func (s *Store) save() (retErr error) {
	sortItems(s.raws.Items)
	f, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("raw.tomlの書き込みに失敗: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("raw.tomlのクローズに失敗: %w", cerr)
		}
	}()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(s.raws); err != nil {
		return fmt.Errorf("raw.tomlのエンコードに失敗: %w", err)
	}
	return nil
}

// SpriteSheets はスプライトシート一覧を返す
func (s *Store) SpriteSheets() []raw.SpriteSheet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.SpriteSheets
}

// Items はアイテム一覧を返す
func (s *Store) Items() []raw.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Items
}

// Item は指定インデックスのアイテムを返す
func (s *Store) Item(index int) (raw.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Items) {
		return raw.Item{}, fmt.Errorf("アイテムインデックスが範囲外: %d", index)
	}
	return s.raws.Items[index], nil
}

// UpdateItem は指定インデックスのアイテムを更新してファイルに保存する
func (s *Store) UpdateItem(index int, item raw.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Items) {
		return fmt.Errorf("アイテムインデックスが範囲外: %d", index)
	}
	s.raws.Items[index] = item
	return s.save()
}

// AddItem は新しいアイテムを追加してファイルに保存する
func (s *Store) AddItem(item raw.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Items = append(s.raws.Items, item)
	return s.save()
}

// DeleteItem は指定インデックスのアイテムを削除してファイルに保存する
func (s *Store) DeleteItem(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Items) {
		return fmt.Errorf("アイテムインデックスが範囲外: %d", index)
	}
	s.raws.Items = append(s.raws.Items[:index], s.raws.Items[index+1:]...)
	return s.save()
}
