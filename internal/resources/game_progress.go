package resources

// イベントID定数
const (
	EventAllCleared = "all_cleared" // 全ダンジョンクリア
)

// EventState はイベントの状態を表す
type EventState struct {
	Active bool // 発火条件を満たしている
	Seen   bool // 視聴済み
}

// GameProgress はゲーム進行に関する永続データを保持する。
// ダンジョンクリアフラグなど、冒険をまたいで残るデータを管理する。
type GameProgress struct {
	ClearedDungeons map[string]bool
	Events          map[string]EventState
}

// NewGameProgress は初期化された GameProgress を返す
func NewGameProgress() *GameProgress {
	return &GameProgress{
		ClearedDungeons: make(map[string]bool),
		Events:          make(map[string]EventState),
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

// IsAllCleared は指定された全ダンジョンがクリア済みかを返す
func (gp *GameProgress) IsAllCleared(dungeonNames []string) bool {
	for _, name := range dungeonNames {
		if !gp.ClearedDungeons[name] {
			return false
		}
	}
	return true
}

// SetEventActive はイベントをActive状態にする
func (gp *GameProgress) SetEventActive(eventID string) {
	ev := gp.Events[eventID]
	ev.Active = true
	gp.Events[eventID] = ev
}

// MarkEventSeen はイベントを視聴済みにする
func (gp *GameProgress) MarkEventSeen(eventID string) {
	ev := gp.Events[eventID]
	ev.Seen = true
	gp.Events[eventID] = ev
}

// IsEventUnseen はイベントが未視聴かを返す
func (gp *GameProgress) IsEventUnseen(eventID string) bool {
	ev, ok := gp.Events[eventID]
	return ok && ev.Active && !ev.Seen
}
