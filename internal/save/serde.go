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
	Version    string          `json:"version"`
	Timestamp  time.Time       `json:"timestamp"`
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
		ecs.C[gc.GameLog](),            // sync.Mutex を含むため不可。毎ロード初期化
		ecs.C[gc.VisualEffects](),      // interfaceスライス・毎フレーム再生成
		ecs.C[gc.Position](),           // GridElementから毎フレーム算出
		ecs.C[gc.StateChangeRequest](), // イベント・毎ターン消費
		ecs.C[gc.StatsChanged](),       // ダーティフラグ
		ecs.C[gc.WeightDirty](),        // ダーティフラグ
		ecs.C[gc.Dead](),               // 一時・毎ターン掃除
		ecs.C[gc.Activity](),           // 実行中アクティビティ・毎ターン変動
		ecs.C[gc.LastActivity](),       // ターン進行で消費
	}
}

// serializeWorld はワールドをark-serdeでJSON化する。一時状態とリソースは除外する
func serializeWorld(world w.World) ([]byte, error) {
	return arkserde.Serialize(world.World,
		arkserde.Opts.SkipComponents(skipComponents()...),
		arkserde.Opts.SkipAllResources(),
	)
}

// deserializeWorld はJSONからワールドを復元する。呼び出し前にworldはReset済みであること
func deserializeWorld(world w.World, worldJSON []byte) error {
	return arkserde.Deserialize(worldJSON, world.World,
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
	q := ecs.NewFilter1[gc.GameProgress](world.World).Query()
	for q.Next() {
		if !found {
			singleton = q.Entity()
			found = true
		}
	}
	if !found {
		return fmt.Errorf("復元データにシングルトン（GameProgress保持エンティティ）が存在しません")
	}
	world.Resources.SingletonEntity = singleton

	world.Components.GameLog.Add(singleton, &gc.GameLog{
		Store: gamelog.NewSafeSlice(gamelog.GameLogMaxSize),
	})
	world.Components.SpatialIndex.Add(singleton, gc.NewSpatialIndex())

	// json:"-"で除外された視界マップを初期化する
	if world.Components.DungeonState.Has(singleton) {
		d := world.Components.DungeonState.Get(singleton)
		if d.ExploredTiles == nil {
			d.ExploredTiles = make(map[gc.GridElement]bool)
		}
		if d.VisibleTiles == nil {
			d.VisibleTiles = make(map[gc.GridElement]bool)
		}
	}
	return nil
}

// extractPlayerName はワールドからプレイヤー名を取得する。存在しなければ空文字を返す。
// クエリを途中でreturnするとワールドがロックされたままになるため、必ず最後まで反復する
func extractPlayerName(world w.World) string {
	name := ""
	q := ecs.NewFilter1[gc.Player](world.World).Query()
	for q.Next() {
		entity := q.Entity()
		if name == "" && world.Components.Name.Has(entity) {
			name = world.Components.Name.Get(entity).Name
		}
	}
	return name
}

// checksumOf は改ざん検知用にチェックサムを除いた封筒のSHA-256を計算する。
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
		panic(fmt.Sprintf("チェックサム計算用のMarshalに失敗: %v", err))
	}
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}
