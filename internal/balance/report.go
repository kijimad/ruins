package balance

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// GenerateReport はマスターデータからシミュレーションを実行し、レポートを生成する。
func GenerateReport(master rawschema.Raws, playerName string, weaponName string, maxDepth int, trials int, seed uint64) (*Report, error) {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil, fmt.Errorf("failed to load player: %w", err)
	}

	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return nil, fmt.Errorf("failed to load weapon: %w", err)
	}

	report := &Report{
		Player: PlayerInfo{
			Name:      playerName,
			Hp:        player.HP,
			Strength:  player.Strength,
			Sensation: player.Sensation,
			Dexterity: player.Dexterity,
			Agility:   player.Agility,
			Defense:   player.Defense,
		},
		Weapon: WeaponInfo{
			Name:     weaponName,
			Damage:   weapon.Damage,
			Accuracy: weapon.Accuracy,
		},
		EnemyTables: []EnemyTableRun{},
	}

	for _, table := range raw.PtrSlice(master.EnemyTables) {
		stats := RunSimulations(master, table.Id, player, weapon, maxDepth, trials, seed)

		run := EnemyTableRun{
			Name:        table.Name,
			MaxDepth:    maxDepth,
			Trials:      trials,
			MedianDepth: stats.MedianDepth(),
			DeathRate:   stats.DeathRate(),
			Depths:      []DepthStat{},
		}

		for depth := 1; depth <= maxDepth; depth++ {
			hps := stats.HPAtDepth(depth)
			if len(hps) == 0 {
				break
			}
			run.Depths = append(run.Depths, DepthStat{
				Depth:              depth,
				MedianHP:           stats.MedianHP(depth),
				P5HP:               stats.P5HP(depth),
				P95HP:              stats.P95HP(depth),
				MedianHPBeforeHeal: stats.MedianHPBeforeHeal(depth),
				P5HPBeforeHeal:     stats.P5HPBeforeHeal(depth),
				P95HPBeforeHeal:    stats.P95HPBeforeHeal(depth),
				SuddenDeathRate:    stats.SuddenDeathRate(depth),
				MedianWeaponDamage: stats.MedianWeaponDamage(depth),
				P5WeaponDamage:     stats.P5WeaponDamage(depth),
				P95WeaponDamage:    stats.P95WeaponDamage(depth),
				MedianKillTurns:    stats.MedianKillTurns(depth),
				P5KillTurns:        stats.P5KillTurns(depth),
				P95KillTurns:       stats.P95KillTurns(depth),
				MedianHunger:       stats.MedianHunger(depth),
				P5Hunger:           stats.P5Hunger(depth),
				P95Hunger:          stats.P95Hunger(depth),
				MedianDamage:       stats.MedianDamagePerFloor(depth, player.HP),
				MedianHealing:      stats.MedianHealingPerFloor(depth),
			})
		}

		report.EnemyTables = append(report.EnemyTables, run)
	}

	// 武器×敵の戦闘メトリクスを生成する
	metrics, err := generateBattleMetrics(master, playerName, seed)
	if err != nil {
		return nil, err
	}
	report.BattleMetrics = metrics

	// 施設種別ごとの loot 分布を生成する
	roomLoot, err := GenerateRoomLoot(master, roomLootTrials, seed)
	if err != nil {
		return nil, err
	}
	report.RoomLoot = roomLoot

	return report, nil
}

// roomLootTrials は施設 loot 集計の試行数。多いほど確率が安定する。
const roomLootTrials = 2000

const battleMetricTrials = 500

// generateBattleMetrics は全武器×全敵の組み合わせで戦闘シミュレーションを実行する
func generateBattleMetrics(master rawschema.Raws, playerName string, seed uint64) ([]BattleMetric, error) {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil, fmt.Errorf("failed to load player for battle metrics: %w", err)
	}

	// 武器一覧を収集する
	type weaponEntry struct {
		name  string
		stats WeaponStats
	}
	var weapons []weaponEntry
	items := raw.PtrSlice(master.Items)
	for i := range items {
		item := &items[i]
		w, err := LoadWeaponFromItem(master, item.Id)
		if err != nil {
			continue
		}
		weapons = append(weapons, weaponEntry{name: item.Name, stats: w})
	}

	// 敵一覧を収集する（全敵テーブルからユニークな敵名を取得する）
	enemySet := make(map[string]struct{})
	for _, table := range raw.PtrSlice(master.EnemyTables) {
		for _, entry := range table.Entries {
			enemySet[entry.Id] = struct{}{}
		}
	}

	// 空でも report.BattleMetrics を null でなく [] にするため空スライスで初期化する
	metrics := []BattleMetric{}
	rng := rand.New(rand.NewPCG(seed, 0))

	for _, w := range weapons {
		for enemyName := range enemySet {
			enemyStats, err := LoadCombatantFromMember(master, enemyName)
			if err != nil {
				continue
			}
			enemyWeapon, err := LoadEnemyWeapon(master, enemyName)
			if err != nil {
				enemyWeapon = WeaponStats{}
			}

			results := RunBattles(player, enemyStats, w.stats, enemyWeapon, battleMetricTrials, rng)
			bs := BattleStats{Results: results}

			metrics = append(metrics, BattleMetric{
				Player:   playerName,
				Weapon:   w.name,
				Enemy:    enemyName,
				Dps:      bs.DPS(),
				IsRanged: w.stats.IsRanged,
			})
		}
	}

	return metrics, nil
}
