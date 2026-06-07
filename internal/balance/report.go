package balance

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// Report はシミュレーション結果のJSON構造
type Report struct {
	Mode          string          `json:"mode"`
	Player        *PlayerInfo     `json:"player,omitempty"`
	Weapon        *WeaponInfo     `json:"weapon,omitempty"`
	EnemyTables   []EnemyTableRun `json:"enemyTables,omitempty"`
	BattleMetrics []BattleMetric  `json:"battleMetrics,omitempty"`
}

// BattleMetric は武器×敵の戦闘シミュレーション結果
type BattleMetric struct {
	Player   string  `json:"player"`
	Weapon   string  `json:"weapon"`
	Enemy    string  `json:"enemy"`
	DPS      float64 `json:"dps"`
	IsRanged bool    `json:"isRanged"`
}

// PlayerInfo はプレイヤーのステータス情報
type PlayerInfo struct {
	Name      string `json:"name"`
	HP        int    `json:"hp"`
	Strength  int    `json:"strength"`
	Sensation int    `json:"sensation"`
	Dexterity int    `json:"dexterity"`
	Agility   int    `json:"agility"`
	Defense   int    `json:"defense"`
}

// WeaponInfo は武器の情報
type WeaponInfo struct {
	Name     string `json:"name"`
	Damage   int    `json:"damage"`
	Accuracy int    `json:"accuracy"`
}

// EnemyTableRun は1つの敵テーブルに対するシミュレーション結果
type EnemyTableRun struct {
	Name        string        `json:"name"`
	MaxDepth    int           `json:"maxDepth"`
	Trials      int           `json:"trials"`
	MedianDepth int           `json:"medianDepth"`
	DeathRate   float64       `json:"deathRate"`
	Depths      []DepthStat   `json:"depths"`
	TrialData   []TrialResult `json:"trialData,omitempty"`
}

// TrialResult は1試行の結果
type TrialResult struct {
	Index        int              `json:"index"`
	ReachedDepth int              `json:"reachedDepth"`
	Died         bool             `json:"died"`
	Depths       []TrialDepthStat `json:"depths"`
}

// TrialDepthStat は1試行の1深度の情報
type TrialDepthStat struct {
	Depth        int    `json:"depth"`
	HP           int    `json:"hp"`
	HPBeforeHeal int    `json:"hpBeforeHeal"`
	Weapon       string `json:"weapon"`
	Hunger       int    `json:"hunger"`
}

// DepthStat は1深度の統計情報
type DepthStat struct {
	Depth              int     `json:"depth"`
	MedianHP           int     `json:"medianHP"`
	P5HP               int     `json:"p5HP"`
	P95HP              int     `json:"p95HP"`
	MedianHPBeforeHeal int     `json:"medianHPBeforeHeal"`
	P5HPBeforeHeal     int     `json:"p5HPBeforeHeal"`
	P95HPBeforeHeal    int     `json:"p95HPBeforeHeal"`
	SuddenDeathRate    float64 `json:"suddenDeathRate"`
	MedianWeaponDamage int     `json:"medianWeaponDamage"`
	P5WeaponDamage     int     `json:"p5WeaponDamage"`
	P95WeaponDamage    int     `json:"p95WeaponDamage"`
	MedianKillTurns    int     `json:"medianKillTurns"`
	P5KillTurns        int     `json:"p5KillTurns"`
	P95KillTurns       int     `json:"p95KillTurns"`
	MedianHunger       int     `json:"medianHunger"`
	P5Hunger           int     `json:"p5Hunger"`
	P95Hunger          int     `json:"p95Hunger"`
	MedianDamage       int     `json:"medianDamage"`
	MedianHealing      int     `json:"medianHealing"`
}

// GenerateReport はマスターデータからシミュレーションを実行し、レポートを生成する
func GenerateReport(master oapi.Raws, playerName string, weaponName string, maxDepth int, trials int, seed uint64) (*Report, error) {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil, fmt.Errorf("プレイヤーのロードに失敗: %w", err)
	}

	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return nil, fmt.Errorf("武器のロードに失敗: %w", err)
	}

	report := &Report{
		Mode: "simple",
		Player: &PlayerInfo{
			Name:      playerName,
			HP:        player.HP,
			Strength:  player.Strength,
			Sensation: player.Sensation,
			Dexterity: player.Dexterity,
			Agility:   player.Agility,
			Defense:   player.Defense,
		},
		Weapon: &WeaponInfo{
			Name:     weaponName,
			Damage:   weapon.Damage,
			Accuracy: weapon.Accuracy,
		},
	}

	for _, table := range raw.PtrSlice(master.EnemyTables) {
		stats := RunSimulations(master, table.Name, player, weapon, maxDepth, trials, seed)

		run := EnemyTableRun{
			Name:        table.Name,
			MaxDepth:    maxDepth,
			Trials:      trials,
			MedianDepth: stats.MedianDepth(),
			DeathRate:   stats.DeathRate(),
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

		for i, r := range stats.Results {
			trial := TrialResult{
				Index:        i,
				ReachedDepth: r.ReachedDepth,
				Died:         r.Died,
			}
			for depth := 1; depth <= r.ReachedDepth; depth++ {
				td := TrialDepthStat{Depth: depth}
				if hp, ok := r.HPByDepth[depth]; ok {
					td.HP = hp
				}
				if hp, ok := r.HPBeforeHealByDepth[depth]; ok {
					td.HPBeforeHeal = hp
				}
				if w, ok := r.WeaponByDepth[depth]; ok {
					td.Weapon = w
				}
				if h, ok := r.HungerByDepth[depth]; ok {
					td.Hunger = h
				}
				trial.Depths = append(trial.Depths, td)
			}
			run.TrialData = append(run.TrialData, trial)
		}

		report.EnemyTables = append(report.EnemyTables, run)
	}

	// 武器×敵の戦闘メトリクスを生成する
	report.BattleMetrics = generateBattleMetrics(master, playerName, seed)

	return report, nil
}

const battleMetricTrials = 500

// generateBattleMetrics は全武器×全敵の組み合わせで戦闘シミュレーションを実行する
func generateBattleMetrics(master oapi.Raws, playerName string, seed uint64) []BattleMetric {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil
	}

	// 武器一覧を収集する
	type weaponEntry struct {
		name  string
		stats WeaponStats
	}
	var weapons []weaponEntry
	for _, item := range raw.PtrSlice(master.Items) {
		w, err := LoadWeaponFromItem(master, item.Name)
		if err != nil {
			continue
		}
		weapons = append(weapons, weaponEntry{name: item.Name, stats: w})
	}

	// 敵一覧を収集する（全敵テーブルからユニークな敵名を取得する）
	enemySet := make(map[string]struct{})
	for _, table := range raw.PtrSlice(master.EnemyTables) {
		for _, entry := range table.Entries {
			enemySet[entry.EnemyName] = struct{}{}
		}
	}

	var metrics []BattleMetric
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
				DPS:      bs.DPS(),
				IsRanged: w.stats.IsRanged,
			})
		}
	}

	return metrics
}

// MarshalJSON はレポートをJSON形式にシリアライズする
func (r *Report) MarshalJSON() ([]byte, error) {
	type Alias Report
	return json.MarshalIndent((*Alias)(r), "", "  ")
}
