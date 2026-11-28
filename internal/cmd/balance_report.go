package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kijimaD/ruins/internal/balance"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
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

// areaSeeds と areaDepths は、拠点のエリアが何日もつかを測る種と北の深さ。種を固定して、同じ報告を作り直せるようにする
var (
	areaSeeds  = []uint64{1, 2, 3, 4, 5}
	areaDepths = []consts.Chunk{0, 40, 80, 120}
)

// measureAreaSupply は種と北の深さごとに、拠点のエリアだけを持つ world を作って食べ物と燃料を数える
func measureAreaSupply() ([]balance.AreaSupplySample, error) {
	var samples []balance.AreaSupplySample
	for _, depth := range areaDepths {
		for _, seed := range areaSeeds {
			world, err := newOverworldWorld()
			if err != nil {
				return nil, err
			}
			if err := overworld.GenerateBandAt(world, seed, depth); err != nil {
				return nil, err
			}
			samples = append(samples, balance.AreaSupplySample{Seed: seed, Depth: depth, Supply: balance.MeasureAreaSupply(world)})
		}
	}
	return samples, nil
}

// runBalanceReport は I/O 副作用を持つ CLI 層で、意図的に単体テストを持たない。導出ロジックは
// balance パッケージ側でテストする。
func runBalanceReport(_ context.Context, _ *cli.Command) error {
	// パスは埋め込み assets FS 基準。ルートが assets/ を剥がした位置なので assets/ 接頭辞は付けない
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	if err != nil {
		return fmt.Errorf("failed to load raw.toml: %w", err)
	}
	md, err := balance.RenderBaselineMarkdown(master, balance.BaselinePlayer, raw.BareHands, balance.BaselineDays)
	if err != nil {
		return fmt.Errorf("failed to render baseline: %w", err)
	}
	samples, err := measureAreaSupply()
	if err != nil {
		return fmt.Errorf("failed to measure area supply: %w", err)
	}
	md += balance.RenderAreaSupplyMarkdown(balance.DefaultParams(master), samples)
	if err := os.WriteFile(reportPath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", reportPath, err)
	}
	fmt.Printf("wrote %s\n", reportPath)
	return nil
}
