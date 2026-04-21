package resources

// GameProgress はゲーム進行に関する永続データを保持する。
// ダンジョンクリアフラグなど、冒険をまたいで残るデータを管理する。
type GameProgress struct {
	ClearedDungeons map[string]bool
}

// NewGameProgress は初期化された GameProgress を返す
func NewGameProgress() *GameProgress {
	return &GameProgress{
		ClearedDungeons: make(map[string]bool),
	}
}

// MarkDungeonCleared はダンジョンをクリア済みにする
func (gp *GameProgress) MarkDungeonCleared(dungeonName string) {
	gp.ClearedDungeons[dungeonName] = true
}

// IsDungeonCleared はダンジョンがクリア済みかを返す
func (gp *GameProgress) IsDungeonCleared(dungeonName string) bool {
	return gp.ClearedDungeons[dungeonName]
}
