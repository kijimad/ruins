package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

const (
	// raidEnemyTable は襲撃の敵の種類を引く raw の敵の表
	raidEnemyTable = "raid"
	// raidSpawnMin と raidSpawnMax は中心から湧かせる輪のチェビシェフ距離
	raidSpawnMin consts.Tile = 18
	raidSpawnMax consts.Tile = 24
	// raidPlayerMin はプレイヤーから離して湧かせるチェビシェフ距離
	raidPlayerMin = 18
	// raidBase と raidPerDanger は1晩の数を危険度から決める
	raidBase      = 12
	raidPerDanger = 3
	// raidMaxPerNight は1晩の数の上限
	raidMaxPerNight = 60
)

// RaidSystem は周期ごとの襲撃の夜を進める。襲撃の日の夜明けに予告し、波のターンに中心を囲んで敵を湧かせる。
// 湧いた敵の動き方は rawschema.Converge のまま変えず、夜が明けても錨を狙い続ける。予告はどこでも出し、波は地上が
// 現ステージのときだけ湧かせる
type RaidSystem struct{}

// String はシステム名を返す
func (sys *RaidSystem) String() string {
	return "RaidSystem"
}

// Update は終わろうとしているターンの経過ターンで、予告と波のどちらに当たるかを見る
func (sys *RaidSystem) Update(world w.World) error {
	turn := query.GetGameTime(world).TotalTurns
	// 予告はどのステージにいても出す。地上へ戻った夜に前触れなく波が来ないように
	if query.RaidWarnsAt(turn) {
		gamelog.New(query.GetGameLog(world)).
			Markup(gamelog.Tag("system", query.T(world, "Something is gathering in the snow. They will come tonight."))).
			Log()
	}
	if !query.IsOnOverworld(world) {
		return nil
	}
	if wave, ok := query.RaidWaveAt(turn); ok {
		return sys.spawnWave(world, wave)
	}
	return nil
}

// spawnWave は wave 番目の波を、query.RaidTargets の最初の的を中心に湧かせ、寝ているプレイヤーを起こす
func (sys *RaidSystem) spawnWave(world w.World, wave int) error {
	targets := query.RaidTargets(world)
	if len(targets) == 0 {
		return nil
	}
	center := targets[0]
	danger := raidDanger(world, center)
	count := raidWaveSize(raidNightSize(danger), wave)
	table, err := raw.GetEnemyTable(world.Resources.RawMaster, raidEnemyTable)
	if err != nil {
		return err
	}
	// 湧かせる前の世界で、的まで道がつながるマスを選ぶ。手番の地図は敵の手番の初めに作り直される
	query.InvalidateDistanceMap(world)
	tiles := raidSpawnTiles(world, query.GetDistanceMap(world), center, count)
	rng := world.Resources.Config.RNG
	spawned := 0
	for _, tile := range tiles {
		name, err := raw.SelectEnemyByWeight(table, rng, int(danger))
		if err != nil {
			return err
		}
		if name == "" {
			continue
		}
		enemy, err := lifecycle.SpawnEnemy(world, tile, name)
		if err != nil {
			return err
		}
		if world.Components.SoloAI.Has(enemy) {
			world.Components.SoloAI.Get(enemy).Movement = rawschema.Converge
		}
		spawned++
	}
	if spawned == 0 {
		return nil
	}
	gamelog.New(query.GetGameLog(world)).
		Markup(gamelog.Tag("system", query.T(world, "They are coming out of the dark."))).
		Log()
	// 眠りは近くに敵が見えると覚めるが、波は視界の外に湧くので、寄ってからでは支度の時間が無い。眠り以外の作業は止めない
	if player, err := query.GetPlayerEntity(world); err == nil && world.Components.Sleeping.Has(player) {
		activity.CancelActivity(player, "woke up to the raid", world)
	}
	return nil
}

// raidDanger は中心の北の深さから危険度を返す
func raidDanger(world w.World, center consts.Coord[consts.Tile]) consts.Danger {
	sb := query.GetSeamlessBand(world)
	if sb == nil || sb.ChunkH <= 0 {
		return consts.MinDanger
	}
	return query.DangerLevelForDepth(consts.NorthDepth(sb.SpawnChunkY(), sb.AbsChunkRow(center.Y)))
}

// raidNightSize は危険度 danger の1晩の数を返す
func raidNightSize(danger consts.Danger) int {
	return min(raidBase+raidPerDanger*int(danger), raidMaxPerNight)
}

// raidWaveSize は1晩の数 total を波の数で分けた、wave 番目の波の数を返す。割り切れない分は後の波に回す
func raidWaveSize(total, wave int) int {
	n := total / query.RaidWaveCount
	if wave >= query.RaidWaveCount-total%query.RaidWaveCount {
		n++
	}
	return n
}

// raidSpawnTiles は中心を囲む輪の上から、湧かせるマスを count 個まで選ぶ。プレイヤーの視界の外で、プレイヤーからも
// raidPlayerMin 以上離れ、距離の地図で中心まで道がつながり、通れて誰もいないマスに限る。プレイヤーから離すのは、
// 暗がりや壁の裏でプレイヤーの隣に湧かせず、群れが寄ってくるのを見せるため。足りなければ輪を外へ広げ、地図の端まで
// 広げても足りなければ見つかった分だけ返す
func raidSpawnTiles(world w.World, m *gc.DistanceMap, center consts.Coord[consts.Tile], count int) []consts.Coord[consts.Tile] {
	si := query.GetSpatialIndex(world)
	if si == nil || count <= 0 {
		return nil
	}
	visible := query.GetVisionState(world).VisibleTiles
	player, err := query.GetPlayerEntity(world)
	hasPlayer := err == nil && world.Components.GridElement.Has(player)
	var playerAt consts.Coord[consts.Tile]
	if hasPlayer {
		playerAt = world.Components.GridElement.Get(player).Coord
	}
	rng := world.Resources.Config.RNG
	var picked []consts.Coord[consts.Tile]
	limit := max(si.MapWidth, si.MapHeight)
	for inner, outer := raidSpawnMin, raidSpawnMax; inner <= limit && len(picked) < count; inner, outer = outer+1, outer+raidSpawnMax-raidSpawnMin+1 {
		var ring []consts.Coord[consts.Tile]
		for y := max(center.Y-outer, 0); y <= min(center.Y+outer, si.MapHeight-1); y++ {
			// 内側の四角に掛かる行は、左右の帯だけを回して内側を調べ直さない
			innerRow := y > center.Y-inner && y < center.Y+inner
			for x := max(center.X-outer, 0); x <= min(center.X+outer, si.MapWidth-1); x++ {
				if innerRow && x > center.X-inner && x < center.X+inner {
					x = center.X + inner - 1
					continue
				}
				c := consts.Coord[consts.Tile]{X: x, Y: y}
				if _, ok := m.Dist(c); !ok {
					continue
				}
				if hasPlayer && geometry.ChebyshevDistance(c, playerAt) < raidPlayerMin {
					continue
				}
				if visible[gc.GridElement{Coord: c}] || !activity.CanMoveTo(world, c, c, ecs.Entity{}) {
					continue
				}
				ring = append(ring, c)
			}
		}
		rng.Shuffle(len(ring), func(i, j int) { ring[i], ring[j] = ring[j], ring[i] })
		picked = append(picked, ring[:min(len(ring), count-len(picked))]...)
	}
	return picked
}
