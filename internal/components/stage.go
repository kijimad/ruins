package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
)

// overworldStageName はオーバーワールド帯ステージの固定名。ダンジョン定義 DungeonOverworld.Name と
// 一致させ、CurrentStage.Name から定義を引けるようにする。
//
// あえて非公開にする。公開する overworld の identity は型付きの NewOverworldStage() だけにし、
// 素の名前を外へ出さない。名前を公開すると CurrentStage.Name == 名前 のような場所判定を誘発するが、
// それは廃した反パターン。外部からは名前で比較できないよう型で塞ぐ。
const overworldStageName = "オーバーワールド"

// StageKey はステージを一意に識別する serde 安全な identity。共存する各ステージのエンティティ同定に使う。
// オーバーワールドは深度0、ダンジョン階は深度1以上。比較可能な値だけで構成する。
//
// あえて interface でなく比較可能な struct にする。StageBound・PortalConnection・Dungeon に埋め込まれる
// 保存対象で、arkserde は interface を保存できず、共存機構は Key == key の等値比較に依存するため、
// identity は素の値でなければならない。オーバーワールドとダンジョンの振る舞い・設定の多態は、Name で
// 引く StageDefinition(OverworldDefinition/DungeonDefinition)が担う。ここは identity だけを持ち種別で分岐しない。
// Name/Depth はステージ同定と導出、定義の引き当て・階数にのみ使い、場所判定には使わない。
type StageKey struct {
	// Name はステージ定義名を保持する。オーバーワールドは NewOverworldStage() が付ける固定名、
	// ダンジョン階は進入先を区別する定義名。実ステージは必ず名前を持つ。ゼロ値のみ未設定として許す。
	//
	// Name で場所を判定しないこと。「今オーバーワールドにいるか」は保有データで判定する
	// query.IsOnOverworld を使う。Name は定義の引き当てとステージ同定にのみ用いる。
	Name string
	// Depth は階の深度を表す。オーバーワールドは 0、ダンジョン階は 1 以上
	Depth int
}

// NewOverworldStage はオーバーワールド帯のステージキーを返す。深度0。
// オーバーワールドの identity を得る唯一の公開手段。場所判定でなくステージの同定・束縛に使う。
func NewOverworldStage() StageKey { return StageKey{Name: overworldStageName} }

// NewDungeonStage は定義 name・深度 depth のダンジョン階のステージキーを返す。
// ダンジョン階は進入先を区別するため定義名を必ず持たせる。名前なしのダンジョン階は作らない。
func NewDungeonStage(name string, depth int) StageKey {
	return StageKey{Name: name, Depth: depth}
}

// Validate はステージキーの整合を検査する。ロード直後など信頼できない入力に使う。
// オーバーワールドは深度0、それ以外の実ステージは深度1以上、という不変条件を守らせる。
func (k StageKey) Validate() error {
	// ゼロ値はどのステージにも属さない未設定として許容する。町にいるときの
	// CurrentStage などステージ未割り当ての状態が正当にありうる
	if k == (StageKey{}) {
		return nil
	}
	if k.Name == overworldStageName {
		if k.Depth != 0 {
			return fmt.Errorf("オーバーワールドステージの深度が不正: %d", k.Depth)
		}
		return nil
	}
	// オーバーワールド以外の実ステージはダンジョン階。定義名を必ず持ち、深度は1以上とする。
	// 名前なしのダンジョン階は作らないため、ここへ来た空名は破損とみなして弾く
	if k.Name == "" {
		return fmt.Errorf("ダンジョンステージに定義名がありません: 深度%d", k.Depth)
	}
	if k.Depth < 1 {
		return fmt.Errorf("ダンジョンステージの深度が不正: %d", k.Depth)
	}
	return nil
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

// DungeonEntrance は遺跡入口プロップが、どの遺跡定義へ入るかを保持する。
// 相互作用 InteractionDungeonEnter の発動時に DefinitionName を読んで進入先を決める。
type DungeonEntrance struct {
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
