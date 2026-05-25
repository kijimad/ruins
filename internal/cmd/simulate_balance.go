package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kijimaD/ruins/internal/balance"
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
		return fmt.Errorf("raw.tomlの読み込みに失敗: %w", err)
	}

	report, err := balance.GenerateReport(&master, "Ash", "素手", simMaxDepth, simTrials, simSeed)
	if err != nil {
		return err
	}

	data, err := report.MarshalJSON()
	if err != nil {
		return fmt.Errorf("JSONのシリアライズに失敗: %w", err)
	}

	outputPath := "balance.json"
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("ファイルの書き込みに失敗: %w", err)
	}

	return nil
}
