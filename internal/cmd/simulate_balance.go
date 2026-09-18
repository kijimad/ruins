package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kijimaD/ruins/internal/balance"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/urfave/cli/v3"
)

// CmdSimulateBalance はバランスシミュレーションを実行してJSON出力するコマンド
var CmdSimulateBalance = &cli.Command{
	Name:        "simulate-balance",
	Usage:       "simulate-balance",
	Description: "Run balance simulation and output results as JSON",
	Action:      runSimulateBalance,
}

const (
	simMaxDepth = 20
	simTrials   = 1000
	simSeed     = 42
)

func runSimulateBalance(_ context.Context, _ *cli.Command) error {
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	if err != nil {
		return fmt.Errorf("failed to load raw.toml: %w", err)
	}

	report, err := balance.GenerateReport(master, "ash", "bare_hands", simMaxDepth, simTrials, simSeed)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %w", err)
	}

	outputPath := "balance.json"
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	printRunIncomeSummary(master)
	return nil
}

// printRunIncomeSummary は生存を織り込んだ loot 手取り収入の要約を stdout に出す。
// 早死にするランほど収入が少ないので、閉形式の期待収入より低く出る。乱数なしの baseline.md には
// 載せず、モンテカルロの C 層指標としてここで見せる。要約の失敗は本処理を止めない。
func printRunIncomeSummary(master oapi.Raws) {
	player, err := balance.LoadCombatantFromMember(master, "ash")
	if err != nil {
		return
	}
	weapon, err := balance.LoadWeaponFromItem(master, "bare_hands")
	if err != nil {
		return
	}
	stats := balance.RunSimulations(master, "ruins_area", player, weapon, simMaxDepth, simTrials, simSeed)
	fmt.Printf("ruins_area %d trials: median depth=%d, median loot income=%d (p10=%d, p90=%d)\n",
		simTrials, stats.MedianDepth(), stats.MedianLootIncome(),
		stats.LootIncomePercentile(0.1), stats.LootIncomePercentile(0.9))
}
