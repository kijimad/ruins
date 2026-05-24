package balance

import (
	"encoding/json"
	"fmt"

	"github.com/kijimaD/ruins/internal/raw"
)

// BalanceReport はシミュレーション結果のJSON構造
type BalanceReport struct {
	Mode        string          `json:"mode"`
	Player      *PlayerInfo     `json:"player,omitempty"`
	Weapon      *WeaponInfo     `json:"weapon,omitempty"`
	EnemyTables []EnemyTableRun `json:"enemyTables,omitempty"`
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
	Name        string      `json:"name"`
	MaxDepth    int         `json:"maxDepth"`
	Trials      int         `json:"trials"`
	MedianDepth int         `json:"medianDepth"`
	DeathRate   float64     `json:"deathRate"`
	Depths      []DepthStat `json:"depths"`
}

// DepthStat は1深度の統計情報
type DepthStat struct {
	Depth              int            `json:"depth"`
	MedianHP           int            `json:"medianHP"`
	P5HP               int            `json:"p5HP"`
	P95HP              int            `json:"p95HP"`
	MedianHPBeforeHeal int            `json:"medianHPBeforeHeal"`
	P5HPBeforeHeal     int            `json:"p5HPBeforeHeal"`
	P95HPBeforeHeal    int            `json:"p95HPBeforeHeal"`
	SuddenDeathRate    float64        `json:"suddenDeathRate"`
	WeaponDistribution map[string]int `json:"weaponDistribution,omitempty"`
}

// GenerateReport はマスターデータからシミュレーションを実行し、レポートを生成する
func GenerateReport(master *raw.Master, playerName string, weaponName string, maxDepth int, trials int, seed uint64) (*BalanceReport, error) {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil, fmt.Errorf("プレイヤーのロードに失敗: %w", err)
	}

	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return nil, fmt.Errorf("武器のロードに失敗: %w", err)
	}

	report := &BalanceReport{
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

	for _, table := range master.Raws.EnemyTables {
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
				WeaponDistribution: stats.WeaponDistribution(depth),
			})
		}

		report.EnemyTables = append(report.EnemyTables, run)
	}

	return report, nil
}

// MarshalJSON はレポートをJSON形式にシリアライズする
func (r *BalanceReport) MarshalJSON() ([]byte, error) {
	type Alias BalanceReport
	return json.MarshalIndent((*Alias)(r), "", "  ")
}
