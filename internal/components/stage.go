package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
)

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
// 比較可能な値だけで構成し、StageBound のフィールドや現在ステージ指標として等値比較する
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
		// ダンジョンは1階以上の深度を持つが遺跡名は持たない。深度0はオーバーワールドの領分
		if k.Ruin != "" {
			return fmt.Errorf("ダンジョンステージに遺跡名がある: %q", k.Ruin)
		}
		if k.Depth < 1 {
			return fmt.Errorf("ダンジョンステージの深度が不正: %d", k.Depth)
		}
		return nil
	case StageKindRuin:
		// 遺跡は遺跡名を必須とし、1階以上の深度を持つ
		if k.Ruin == "" {
			return fmt.Errorf("遺跡ステージに遺跡名がない")
		}
		if k.Depth < 1 {
			return fmt.Errorf("遺跡ステージの深度が不正: %d", k.Depth)
		}
		return nil
	}
	return fmt.Errorf("未知のステージ種別: %q", k.Kind)
}

// StageBound はエンティティがどのステージに束縛され、そのライフサイクルを共にするかを表す。
// これを持つエンティティは、ステージが退避されれば Suspended になり、完全離脱で一括除去される。
// 地形・敵・アイテムなどステージ固有で作り直せるものが対象。
//
// Player・SquadMember・共有シングルトンには付けない。これらはステージを渡り歩く訪問者で、
// どのステージとも運命を共にしない。束縛しないことで suspend/purge/resume のどの操作からも
// 自動で外れる。プレイヤーの現在地は Dungeon.CurrentStage が持つ。
type StageBound struct {
	// Key は束縛先ステージを保持する
	Key StageKey
}

// PortalConnection はポータルの行き先を保持する。触れると Stage へ swapTo し Coord へ配置する。
// 生成時に往復の両端を相互結線する。findPortalPosition の探索を置き換え、遺跡の複数入口でも
// どのポータルがどこへ繋がるかが曖昧にならない。Stage・Coord とも比較可能で serde 対象。
type PortalConnection struct {
	// Stage は行き先ステージ
	Stage StageKey
	// Coord は行き先ステージ内の着地座標
	Coord consts.Coord[consts.Tile]
}

// RuinEntrance は遺跡入口プロップが、どの遺跡定義へ入るかを保持する。
// 相互作用 InteractionRuinEnter の発動時に DefinitionName を読んで進入先を決める。
type RuinEntrance struct {
	// DefinitionName は進入する遺跡の定義名
	DefinitionName string
}

// Suspended は現ステージ以外に属し、現在のフレームで稼働しないことを表すマーカー。
// ステージ跨ぎのシステムは Without(Suspended) で現ステージだけを処理する。
//
// 稼働を既定にするための否定形マーカー。新しい湧きやプレイヤー・隊員は Suspended を
// 持たず、何もせずとも既定で稼働する。肯定形 Active にすると、湧きや訪問者へ付け忘れた
// ときに不可視という重い失敗になり、湧くたびに付与する責務も生じる。
// 稼働を既定にするには非稼働側を印すのが正しい。
type Suspended struct{}
