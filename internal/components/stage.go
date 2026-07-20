package components

import "fmt"

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

// NewOverworldStage はオーバーワールド帯のステージキーを返す。
func NewOverworldStage() StageKey { return StageKey{Kind: StageKindOverworld} }

// NewDungeonStage は深度 depth の通常ダンジョン階のステージキーを返す。
func NewDungeonStage(depth int) StageKey {
	return StageKey{Kind: StageKindDungeon, Depth: depth}
}

// NewRuinStage は遺跡定義 name・深度 depth の遺跡階のステージキーを返す。
func NewRuinStage(name string, depth int) StageKey {
	return StageKey{Kind: StageKindRuin, Ruin: name, Depth: depth}
}

// Validate はステージキーの整合を検査する。ロード直後など信頼できない入力に使う。
// Kind ごとに設定してよいフィールドが決まっており、それ以外が埋まっていれば不正とみなす。
// これでコンストラクタを通さず組み立てられた不正なキーを境界で弾ける。
// default を置かず既知種別を網羅させ、未知種別は末尾で loud に error にする
func (k StageKey) Validate() error {
	// ゼロ値はどのステージにも属さない未設定として許容する。町にいるときの
	// CurrentStage などステージ未割り当ての状態が正当にありうる
	if k == (StageKey{}) {
		return nil
	}
	switch k.Kind {
	case StageKindOverworld:
		// オーバーワールドは深度も遺跡名も持たない
		if k.Ruin != "" || k.Depth != 0 {
			return fmt.Errorf("オーバーワールドステージに余分な値がある: Ruin=%q Depth=%d", k.Ruin, k.Depth)
		}
		return nil
	case StageKindDungeon:
		// ダンジョンは深度を持つが遺跡名は持たない
		if k.Ruin != "" {
			return fmt.Errorf("ダンジョンステージに遺跡名がある: %q", k.Ruin)
		}
		return nil
	case StageKindRuin:
		// 遺跡は遺跡名を必須とする
		if k.Ruin == "" {
			return fmt.Errorf("遺跡ステージに遺跡名がない")
		}
		return nil
	}
	return fmt.Errorf("未知のステージ種別: %q", k.Kind)
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
