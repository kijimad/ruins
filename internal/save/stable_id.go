// Package save は安定ID + リフレクションベースのECSシリアライゼーションを提供する
package save

import (
	"fmt"
	"sync"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// StableID は安定エンティティ識別子
type StableID struct {
	Index      uint32 `json:"index"`      // エンティティのインデックス
	Generation uint32 `json:"generation"` // 世代番号（削除/再利用時にインクリメント）
}

// StableIDManager は安定IDの管理を行う
type StableIDManager struct {
	// 実際のエンティティ -> 安定ID
	entityToStable map[ecs.Entity]StableID
	// 安定ID -> 実際のエンティティ
	stableToEntity map[StableID]ecs.Entity
	// インデックス別の世代管理
	generations map[uint32]uint32
	// 次に使用するインデックス
	nextIndex uint32
	// 再利用可能なインデックスのスタック
	freeIndices []uint32
	// 排他制御
	mutex sync.RWMutex
}

// NewStableIDManager は新しいStableIDManagerを作成
func NewStableIDManager() *StableIDManager {
	return &StableIDManager{
		entityToStable: make(map[ecs.Entity]StableID),
		stableToEntity: make(map[StableID]ecs.Entity),
		generations:    make(map[uint32]uint32),
		nextIndex:      1, // 0は無効なIDとして予約
		freeIndices:    make([]uint32, 0),
	}
}

// GetStableID は実際のエンティティから安定IDを取得（なければ新規作成）
func (m *StableIDManager) GetStableID(entity ecs.Entity) StableID {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 既存のマッピングをチェック
	if stableID, exists := m.entityToStable[entity]; exists {
		return stableID
	}

	// 新しい安定IDを作成
	var index uint32
	if len(m.freeIndices) > 0 {
		// 再利用可能なインデックスを使用
		index = m.freeIndices[len(m.freeIndices)-1]
		m.freeIndices = m.freeIndices[:len(m.freeIndices)-1]
	} else {
		// 新しいインデックスを使用
		index = m.nextIndex
		m.nextIndex++
	}

	// 世代番号を取得（初回は0）
	generation := m.generations[index]

	stableID := StableID{
		Index:      index,
		Generation: generation,
	}

	// マッピングを登録
	m.entityToStable[entity] = stableID
	m.stableToEntity[stableID] = entity

	return stableID
}

// GetEntity は安定IDから実際のエンティティを取得
func (m *StableIDManager) GetEntity(stableID StableID) (ecs.Entity, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entity, exists := m.stableToEntity[stableID]
	return entity, exists
}

// RegisterEntity は既存のエンティティに安定IDをマッピング
func (m *StableIDManager) RegisterEntity(entity ecs.Entity, stableID StableID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 世代チェック
	currentGeneration, exists := m.generations[stableID.Index]
	if exists && currentGeneration != stableID.Generation {
		return fmt.Errorf("stable ID generation mismatch: expected %d, got %d",
			currentGeneration, stableID.Generation)
	}

	// マッピング登録
	m.entityToStable[entity] = stableID
	m.stableToEntity[stableID] = entity
	m.generations[stableID.Index] = stableID.Generation

	// nextIndexを更新
	if stableID.Index >= m.nextIndex {
		m.nextIndex = stableID.Index + 1
	}

	return nil
}

// UnregisterEntity はエンティティの登録を解除し、世代をインクリメント
func (m *StableIDManager) UnregisterEntity(entity ecs.Entity) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stableID, exists := m.entityToStable[entity]
	if !exists {
		return
	}

	// マッピングを削除
	delete(m.entityToStable, entity)
	delete(m.stableToEntity, stableID)

	// 世代をインクリメント（同じインデックスの再利用時に区別するため）
	m.generations[stableID.Index]++

	// インデックスを再利用可能にする
	m.freeIndices = append(m.freeIndices, stableID.Index)
}

// IsValid は安定IDが現在有効かどうかをチェック
func (m *StableIDManager) IsValid(stableID StableID) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	currentGeneration, exists := m.generations[stableID.Index]
	if !exists {
		return false
	}

	return currentGeneration == stableID.Generation
}

// Clear は全ての登録をクリア
func (m *StableIDManager) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.entityToStable = make(map[ecs.Entity]StableID)
	m.stableToEntity = make(map[StableID]ecs.Entity)
	m.generations = make(map[uint32]uint32)
	m.nextIndex = 1
	m.freeIndices = make([]uint32, 0)
}

// GetAllStableIDs は登録されている全ての安定IDを取得
func (m *StableIDManager) GetAllStableIDs() []StableID {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stableIDs := make([]StableID, 0, len(m.entityToStable))
	for _, stableID := range m.entityToStable {
		stableIDs = append(stableIDs, stableID)
	}

	return stableIDs
}

// String は安定IDの文字列表現を返す
func (id StableID) String() string {
	return fmt.Sprintf("StableID{%d:%d}", id.Index, id.Generation)
}

// IsNull は安定IDが無効（null）かどうかをチェック
func (id StableID) IsNull() bool {
	return id.Index == 0
}

// NullStableID は無効な安定IDを表す
var NullStableID = StableID{Index: 0, Generation: 0}
