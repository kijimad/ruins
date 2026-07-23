package designdoc

// Status は設計ドキュメントの進行状態を表す。GitHub Issue の open/closed に相当する。
type Status string

const (
	// StatusDraft は下書き。まだ着手していない。
	StatusDraft Status = "draft"
	// StatusAccepted は方針合意済みだが未着手。
	StatusAccepted Status = "accepted"
	// StatusInProgress は実装途中。進捗に未完タスクが残る。
	StatusInProgress Status = "in-progress"
	// StatusDone は完了。進捗が全て済んでいる。
	StatusDone Status = "done"
	// StatusSuperseded は後続ドキュメントに置き換えられた。
	StatusSuperseded Status = "superseded"
	// StatusDropped は不採用として打ち切った。
	StatusDropped Status = "dropped"
)

// Valid は status が定義済みの値かを返す。
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusAccepted, StatusInProgress, StatusDone, StatusSuperseded, StatusDropped:
		return true
	}

	return false
}

// IsOpen は未完了、すなわちバックログとして着手対象かを返す。
// done・superseded・dropped は閉じた状態とみなす。
func (s Status) IsOpen() bool {
	switch s {
	case StatusDraft, StatusAccepted, StatusInProgress:
		return true
	case StatusDone, StatusSuperseded, StatusDropped:
		return false
	}

	return false
}

// Auto はルーチンが無人で着手してよいかの分類を表す。
type Auto string

const (
	// AutoMechanical は意思決定不要の機械的タスク。無人ルーチンの対象にできる。
	AutoMechanical Auto = "mechanical"
	// AutoNeedsDecision は人の判断を要するタスク。無人着手しない。
	AutoNeedsDecision Auto = "needs-decision"
)

// Valid は auto が定義済みの値かを返す。
func (a Auto) Valid() bool {
	switch a {
	case AutoMechanical, AutoNeedsDecision:
		return true
	}

	return false
}

// KnownTags は tags に使う推奨語彙。未知タグは検証で警告する。表記ゆれを抑えるための緩い辞書であり、
// 厳格な制約ではない。
var KnownTags = []string{
	"refactor",   // リファクタリング、責務分割、用語統一
	"ecs",        // Ark ECS、コンポーネント、シングルトン
	"worldgen",   // マップ生成、シームレスワールド、チャンク
	"combat",     // 戦闘、AI行動、遠距離、ボス
	"ui",         // UIテーマ、ゴールデンテスト、HUD
	"item",       // アイテム、収納、スタック、運搬
	"movement",   // 移動、走り、マクロ移動
	"save",       // セーブ、永続化、ユーザー設定
	"ci",         // CI、linter、キャッシュ、コード生成
	"perf",       // パフォーマンス、ベンチマーク、プロファイル
	"gamedesign", // ゲーム方向性、バランス、ラン構造
	"member",     // 隊員、隊マネジメント
	"narrative",  // オープニング、ナレーション、コンセプト
	"steam",      // Steamリリース、ストアグラフィック
	"meta",       // 開発ワークフロー、設計ドキュメント運用
}

// Frontmatter は設計ドキュメント冒頭の YAML メタ情報を表す。
type Frontmatter struct {
	Status Status   `yaml:"status"`
	Tags   []string `yaml:"tags"`
	Auto   Auto     `yaml:"auto"`
}

// Document は1つの設計ドキュメントの解析結果を表す。
type Document struct {
	// Path は docs/design からの、あるいは呼び出し側が渡したファイルパス。
	Path string
	// Number は docs/design/YYYYMMDD_NN.md の NN。ドキュメントの連番。取得できなければ 0。
	Number int
	// Title は本文冒頭の `# ` 見出し。無ければ空。
	Title string
	// Front は解析した frontmatter。HasFront が false のときはゼロ値。
	Front Frontmatter
	// HasFront は frontmatter を持つかどうか。
	HasFront bool
	// HasProgress は `## 進捗` セクションを持つかどうか。
	HasProgress bool
	// OpenTasks は進捗の未完タスク数。`- [ ]`。
	OpenTasks int
	// DoneTasks は進捗の完了タスク数。`- [x]`。
	DoneTasks int
	// SkippedTasks は意図的に着手しないタスク数。`- [~]`。open にも done にも数えない。
	SkippedTasks int
	// Body は frontmatter を除いた本文。
	Body string
}
