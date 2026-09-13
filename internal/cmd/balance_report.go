package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kijimaD/ruins/internal/balance"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/urfave/cli/v3"
)

// CmdBalanceReport は序盤戦闘のベースラインを計算し docs/balance/baseline.md へ出力する。
// 実コードの戦闘式と raw.toml から決定論的に難易度カーブを導出し、目標回廊と突き合わせる。
var CmdBalanceReport = &cli.Command{
	Name:        "balance-report",
	Usage:       "balance-report",
	Description: "Derive the early-combat difficulty curve and write docs/balance/baseline.md. Run from the repository root; paths are relative to it",
	Action:      runBalanceReport,
}

// reportPath は生成先。基準の player/weapon/days は balance の共有定数を使い二重化を避ける。
const reportPath = "docs/balance/baseline.md"

// runBalanceReport は I/O 副作用を持つ CLI 層で、意図的に単体テストを持たない。導出ロジックは
// balance パッケージ側でテストする。
func runBalanceReport(_ context.Context, _ *cli.Command) error {
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	if err != nil {
		return fmt.Errorf("failed to load raw.toml: %w", err)
	}
	md, err := balance.RenderBaselineMarkdown(master, balance.BaselinePlayer, balance.BaselineWeapon, balance.BaselineDays)
	if err != nil {
		return fmt.Errorf("failed to render baseline: %w", err)
	}
	if err := os.WriteFile(reportPath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", reportPath, err)
	}
	fmt.Printf("wrote %s\n", reportPath)
	return nil
}
