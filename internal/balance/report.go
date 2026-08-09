package balance

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// GenerateReport はマスターデータからシミュレーションを実行し、レポートを生成する。返す型は tsp 由来の
// oapi.BalanceReport で、balance.json の形は oas/typespec/balance.tsp を単一ソースとする。
func GenerateReport(master oapi.Raws, playerName string, weaponName string, maxDepth int, trials int, seed uint64) (*oapi.BalanceReport, error) {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil, fmt.Errorf("failed to load player: %w", err)
	}

	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return nil, fmt.Errorf("failed to load weapon: %w", err)
	}

	report := &oapi.BalanceReport{
		Mode: "simple",
		Player: &oapi.BalancePlayerInfo{
			Name:      playerName,
			Hp:        int32(player.HP),
			Strength:  int32(player.Strength),
			Sensation: int32(player.Sensation),
			Dexterity: int32(player.Dexterity),
			Agility:   int32(player.Agility),
			Defense:   int32(player.Defense),
		},
		Weapon: &oapi.BalanceWeaponInfo{
			Name:     weaponName,
			Damage:   int32(weapon.Damage),
			Accuracy: int32(weapon.Accuracy),
		},
		// tsp で required の配列は空でも JSON を null でなく [] にする。nil スライスは null になり、UI の map が
		// フォールバック無しで回せなくなるため、append する配列は空スライスで初期化して契約を守る。
		EnemyTables: []oapi.BalanceEnemyTableRun{},
	}

	for _, table := range raw.PtrSlice(master.EnemyTables) {
		stats := RunSimulations(master, table.Id, player, weapon, maxDepth, trials, seed)

		run := oapi.BalanceEnemyTableRun{
			Name:        table.Name,
			MaxDepth:    int32(maxDepth),
			Trials:      int32(trials),
			MedianDepth: int32(stats.MedianDepth()),
			DeathRate:   stats.DeathRate(),
			Depths:      []oapi.BalanceDepthStat{},
			TrialData:   []oapi.BalanceTrialResult{},
		}

		for depth := 1; depth <= maxDepth; depth++ {
			hps := stats.HPAtDepth(depth)
			if len(hps) == 0 {
				break
			}
			run.Depths = append(run.Depths, oapi.BalanceDepthStat{
				Depth:              int32(depth),
				MedianHP:           int32(stats.MedianHP(depth)),
				P5HP:               int32(stats.P5HP(depth)),
				P95HP:              int32(stats.P95HP(depth)),
				MedianHPBeforeHeal: int32(stats.MedianHPBeforeHeal(depth)),
				P5HPBeforeHeal:     int32(stats.P5HPBeforeHeal(depth)),
				P95HPBeforeHeal:    int32(stats.P95HPBeforeHeal(depth)),
				SuddenDeathRate:    stats.SuddenDeathRate(depth),
				MedianWeaponDamage: int32(stats.MedianWeaponDamage(depth)),
				P5WeaponDamage:     int32(stats.P5WeaponDamage(depth)),
				P95WeaponDamage:    int32(stats.P95WeaponDamage(depth)),
				MedianKillTurns:    int32(stats.MedianKillTurns(depth)),
				P5KillTurns:        int32(stats.P5KillTurns(depth)),
				P95KillTurns:       int32(stats.P95KillTurns(depth)),
				MedianHunger:       int32(stats.MedianHunger(depth)),
				P5Hunger:           int32(stats.P5Hunger(depth)),
				P95Hunger:          int32(stats.P95Hunger(depth)),
				MedianDamage:       int32(stats.MedianDamagePerFloor(depth, player.HP)),
				MedianHealing:      int32(stats.MedianHealingPerFloor(depth)),
			})
		}

		for i, r := range stats.Results {
			trial := oapi.BalanceTrialResult{
				Index:        int32(i),
				ReachedDepth: int32(r.ReachedDepth),
				Died:         r.Died,
				Depths:       []oapi.BalanceTrialDepthStat{},
			}
			for depth := 1; depth <= r.ReachedDepth; depth++ {
				td := oapi.BalanceTrialDepthStat{Depth: int32(depth)}
				if hp, ok := r.HPByDepth[depth]; ok {
					td.Hp = int32(hp)
				}
				if hp, ok := r.HPBeforeHealByDepth[depth]; ok {
					td.HpBeforeHeal = int32(hp)
				}
				if w, ok := r.WeaponByDepth[depth]; ok {
					td.Weapon = w
				}
				if h, ok := r.HungerByDepth[depth]; ok {
					td.Hunger = int32(h)
				}
				trial.Depths = append(trial.Depths, td)
			}
			run.TrialData = append(run.TrialData, trial)
		}

		report.EnemyTables = append(report.EnemyTables, run)
	}

	// 武器×敵の戦闘メトリクスを生成する
	report.BattleMetrics = generateBattleMetrics(master, playerName, seed)

	// 施設種別ごとの loot 分布を生成する
	report.RoomLoot = GenerateRoomLoot(master, roomLootTrials, seed)

	return report, nil
}

// roomLootTrials は施設 loot 集計の試行数。多いほど確率が安定する。
const roomLootTrials = 2000

const battleMetricTrials = 500

// generateBattleMetrics は全武器×全敵の組み合わせで戦闘シミュレーションを実行する
func generateBattleMetrics(master oapi.Raws, playerName string, seed uint64) []oapi.BalanceBattleMetric {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		log.Printf("generateBattleMetrics: failed to load player %q: %v", playerName, err)
		return nil
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
	metrics := []oapi.BalanceBattleMetric{}
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

			metrics = append(metrics, oapi.BalanceBattleMetric{
				Player:   playerName,
				Weapon:   w.name,
				Enemy:    enemyName,
				Dps:      bs.DPS(),
				IsRanged: w.stats.IsRanged,
			})
		}
	}

	return metrics
}
