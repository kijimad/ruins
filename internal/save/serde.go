package save

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	arkserde "github.com/mlange-42/ark-serde"
	"github.com/mlange-42/ark/ecs"
)

// saveEnvelope はセーブファイルの外枠。ark-serde のワールドJSONをメタ情報で包む
type saveEnvelope struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	// Checksum はキーレスSHA-256による破損検知用の値。改ざん検知（攻撃者が
	// world改変後にchecksumを再計算できる）は目的としない
	Checksum   string          `json:"checksum"`
	PlayerName string          `json:"playerName"`
	World      json.RawMessage `json:"world"`
}

// skipComponents はserde除外対象を返す。
// serde非互換（struct-keyed map / interface / mutex）と、毎フレーム・毎ターン
// 再生成される一時状態のみを除外し、地形・敵・アイテムを含む残りは丸ごと保存する。
func skipComponents() []ecs.Comp {
	return []ecs.Comp{
		ecs.C[gc.SpatialIndex](),       // struct-keyed map。ロード時に再構築
		ecs.C[gc.VisionState](),        // struct-keyed map。視界更新で再構築
		ecs.C[gc.GameLog](),            // sync.Mutex を含むため不可。毎ロード初期化
		ecs.C[gc.VisualEffects](),      // interfaceスライス・毎フレーム再生成
		ecs.C[gc.Position](),           // GridElementから毎フレーム算出
		ecs.C[gc.StateChangeRequest](), // イベント・毎ターン消費
		ecs.C[gc.StatsChanged](),       // ダーティフラグ
		ecs.C[gc.WeightDirty](),        // ダーティフラグ
		ecs.C[gc.Dead](),               // 一時・毎ターン掃除
		ecs.C[gc.Activity](),           // 実行中アクティビティ・毎ターン変動
		ecs.C[gc.LastActivity](),       // ターン進行で消費
		ecs.C[gc.UserSettings](),       // config 由来のランタイムミラー。ロード時に config から再構築
	}
}

// serializeWorld はワールドをark-serdeでJSON化する。一時状態とリソースは除外する
func serializeWorld(world w.World) ([]byte, error) {
	return arkserde.Serialize(world.ECS,
		arkserde.Opts.SkipComponents(skipComponents()...),
		arkserde.Opts.SkipAllResources(),
	)
}

// deserializeWorld はJSONからワールドを復元する。呼び出し前にworldはReset済みであること。
//
// セーブファイルは破損しうる信頼境界であり、arkserde/ark は壊れた入力で panic することがある。
// panic を error に変換してゲームのクラッシュを防ぐ。ロード失敗は呼び出し側がエラー表示で扱う。
func deserializeWorld(world w.World, worldJSON []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to restore save data; data may be corrupted: %v", r)
		}
	}()
	return arkserde.Deserialize(worldJSON, world.ECS,
		arkserde.Opts.SkipComponents(skipComponents()...),
		arkserde.Opts.SkipAllResources(),
	)
}

// reestablishSingleton は復元後のシングルトンエンティティを再確立する。
// スキップした一時コンポーネント（GameLog/SpatialIndex）を再付与し、
// json:"-"で除外された視界マップを初期化し、Resourcesの参照を張り直す。
func reestablishSingleton(world w.World) error {
	// GameProgressを持つ最初のエンティティをシングルトンとする。
	// 途中returnはワールドをロックしたまま残すため、クエリは最後まで反復する
	var singleton ecs.Entity
	found := false
	q := ecs.NewFilter1[gc.GameProgress](world.ECS).Query()
	for q.Next() {
		if !found {
			singleton = q.Entity()
			found = true
		}
	}
	if !found {
		return fmt.Errorf("restored data has no singleton entity that holds GameProgress")
	}
	world.Resources.SingletonEntity = singleton

	world.Components.GameLog.Add(singleton, &gc.GameLog{
		Store: gamelog.NewSafeSlice(gamelog.GameLogMaxSize),
	})
	world.Components.SpatialIndex.Add(singleton, gc.NewSpatialIndex())
	// 視界計算の一時状態は serde 除外なのでロード後に再構築する
	world.Components.VisionState.Add(singleton, gc.NewVisionState())
	// グローバル設定は serde 除外なので config から再構築する
	world.Components.UserSettings.Add(singleton, gc.NewUserSettings(world.Resources.Config.User.Language))

	// json:"-"で除外された各ステージの探索履歴を初期化する。入場時リセット方針なので空でよい。
	// ロック中の反復では構造変更しないため、対象を集めてから初期化する
	var metas []ecs.Entity
	mq := ecs.NewFilter1[gc.StageField](world.ECS).Query()
	for mq.Next() {
		metas = append(metas, mq.Entity())
	}
	for _, e := range metas {
		field := world.Components.StageField.Get(e)
		if field.ExploredTiles == nil {
			field.ExploredTiles = make(map[gc.GridElement]bool)
		}
	}

	// CharModifiers.Sources は json:"-" で除外され、ロード時に再計算する規約。
	// 再計算は StatsChangedSystem が StatsChanged ダーティフラグの立ったエンティティにだけ行うが、
	// StatsChanged 自体も serde 除外でロード直後は全消えしている。Abilities を持つ全エンティティに
	// 付け直し、次の更新で Sources が再構築されるようにする。
	// ロック中の反復では構造変更しないため、対象を集めてから付与する
	var statsTargets []ecs.Entity
	aq := ecs.NewFilter1[gc.Abilities](world.ECS).Query()
	for aq.Next() {
		statsTargets = append(statsTargets, aq.Entity())
	}
	for _, e := range statsTargets {
		world.Components.StatsChanged.Add(e, &gc.StatsChanged{})
	}
	return nil
}

// validateStages は復元したステージ関連の値の整合を検査する。
// セーブファイルは信頼境界であり、コンストラクタを通らない不正な StageKey が
// 紛れうる。StageKey を内包する全コンポーネント、Dungeon.CurrentStage・StageBound.Key・
// PortalConnection.Stage を Validate で弾く。ロックを避けるため先にキーを集め、反復を終えてから検証する
func validateStages(world w.World) error {
	var keys []gc.StageKey
	dq := ecs.NewFilter1[gc.Dungeon](world.ECS).Query()
	for dq.Next() {
		keys = append(keys, world.Components.Dungeon.Get(dq.Entity()).CurrentStage)
	}
	mq := ecs.NewFilter1[gc.StageBound](world.ECS).Query()
	for mq.Next() {
		keys = append(keys, world.Components.StageBound.Get(mq.Entity()).Key)
	}
	pq := ecs.NewFilter1[gc.PortalConnection](world.ECS).Query()
	for pq.Next() {
		keys = append(keys, world.Components.PortalConnection.Get(pq.Entity()).Stage)
	}
	for _, k := range keys {
		if err := k.Validate(); err != nil {
			return fmt.Errorf("restored stage key is invalid: %w", err)
		}
	}
	return nil
}

// extractPlayerName はワールドからプレイヤー名を取得する。存在しなければ空文字を返す。
// クエリを途中でreturnするとワールドがロックされたままになるため、必ず最後まで反復する
func extractPlayerName(world w.World) string {
	name := ""
	q := ecs.NewFilter1[gc.Player](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		if name == "" && world.Components.Name.Has(entity) {
			name = world.Components.Name.Get(entity).Name
		}
	}
	return name
}

// checksumOf は破損検知用にチェックサムを除いた封筒のSHA-256を計算する。
// json.Marshal は json.RawMessage を compact するため、保存ファイルが
// MarshalIndent で整形されていても検証時に同一バイト列へ正規化され、値が一致する。
// 封筒は全てJSON互換型のためMarshalは失敗しないが、万一失敗した場合はpanicする
func checksumOf(env *saveEnvelope) string {
	target := saveEnvelope{
		Version:    env.Version,
		Timestamp:  env.Timestamp,
		PlayerName: env.PlayerName,
		World:      env.World,
	}
	jsonBytes, err := json.Marshal(target)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal for checksum: %v", err))
	}
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}
