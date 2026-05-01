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
	sortMembers(s.raws.Members)
	sortRecipes(s.raws.Recipes)
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

// sortMembers はメンバーを名前順にソートする
func sortMembers(members []raw.Member) {
	sort.SliceStable(members, func(i, j int) bool {
		return members[i].Name < members[j].Name
	})
}

// sortRecipes はレシピを名前順にソートする
func sortRecipes(recipes []raw.Recipe) {
	sort.SliceStable(recipes, func(i, j int) bool {
		return recipes[i].Name < recipes[j].Name
	})
}

func (s *Store) save() (retErr error) {
	sortItems(s.raws.Items)
	sortMembers(s.raws.Members)
	sortRecipes(s.raws.Recipes)
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

// Members はメンバー一覧を返す
func (s *Store) Members() []raw.Member {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Members
}

// Member は指定インデックスのメンバーを返す
func (s *Store) Member(index int) (raw.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Members) {
		return raw.Member{}, fmt.Errorf("メンバーインデックスが範囲外: %d", index)
	}
	return s.raws.Members[index], nil
}

// UpdateMember は指定インデックスのメンバーを更新してファイルに保存する
func (s *Store) UpdateMember(index int, member raw.Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Members) {
		return fmt.Errorf("メンバーインデックスが範囲外: %d", index)
	}
	s.raws.Members[index] = member
	return s.save()
}

// AddMember は新しいメンバーを追加してファイルに保存する
func (s *Store) AddMember(member raw.Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Members = append(s.raws.Members, member)
	return s.save()
}

// DeleteMember は指定インデックスのメンバーを削除してファイルに保存する
func (s *Store) DeleteMember(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Members) {
		return fmt.Errorf("メンバーインデックスが範囲外: %d", index)
	}
	s.raws.Members = append(s.raws.Members[:index], s.raws.Members[index+1:]...)
	return s.save()
}

// Recipes はレシピ一覧を返す
func (s *Store) Recipes() []raw.Recipe {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Recipes
}

// Recipe は指定インデックスのレシピを返す
func (s *Store) Recipe(index int) (raw.Recipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Recipes) {
		return raw.Recipe{}, fmt.Errorf("レシピインデックスが範囲外: %d", index)
	}
	return s.raws.Recipes[index], nil
}

// UpdateRecipe は指定インデックスのレシピを更新してファイルに保存する
func (s *Store) UpdateRecipe(index int, recipe raw.Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Recipes) {
		return fmt.Errorf("レシピインデックスが範囲外: %d", index)
	}
	s.raws.Recipes[index] = recipe
	return s.save()
}

// AddRecipe は新しいレシピを追加してファイルに保存する
func (s *Store) AddRecipe(recipe raw.Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Recipes = append(s.raws.Recipes, recipe)
	return s.save()
}

// DeleteRecipe は指定インデックスのレシピを削除してファイルに保存する
func (s *Store) DeleteRecipe(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Recipes) {
		return fmt.Errorf("レシピインデックスが範囲外: %d", index)
	}
	s.raws.Recipes = append(s.raws.Recipes[:index], s.raws.Recipes[index+1:]...)
	return s.save()
}
