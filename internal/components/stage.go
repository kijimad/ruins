package components

// StageKind はステージの種類を表す。往復するステージを種別で区別する
type StageKind string

const (
	// StageKindOverworld はオーバーワールド帯を表す
	StageKindOverworld StageKind = "overworld"
	// StageKindDungeon は通常ダンジョンの階を表す
	StageKindDungeon StageKind = "dungeon"
	// StageKindRuin は遺跡の階を表す
	StageKindRuin StageKind = "ruin"
)

// StageKey はステージを一意に識別する。共存する各ステージのエンティティを同定するのに使う。
// 比較可能な値だけで構成し、StageMember のフィールドや現在ステージ指標として等値比較する
type StageKey struct {
	// Kind はステージの種類を表す
	Kind StageKind
	// Ruin は遺跡定義名を保持する。Kind が StageKindRuin のときだけ設定し、それ以外は空にする
	Ruin string
	// Depth は階の深度を表す。オーバーワールドは 0 とする
	Depth int
}

// StageMember はエンティティが所属するステージを表す。
// 往復で退避されるステージの同定と、完全離脱時の一括除去の対象選択に使う。
// Player・SquadMember・共有シングルトンには付けない
type StageMember struct {
	// Key は所属ステージを保持する
	Key StageKey
}

// Suspended は現ステージ以外に属し、現在のフレームで稼働しないことを表すマーカー。
// ステージ跨ぎのシステムは Without(Suspended) で現ステージだけを処理する
type Suspended struct{}
