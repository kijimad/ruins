package cmd

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"github.com/urfave/cli/v3"
	"golang.org/x/perf/benchfmt"
)

// CmdBenchgate は base と PR のベンチ結果を比較し、退化したパッケージを報告するサブコマンド。
var CmdBenchgate = &cli.Command{
	Name:      "benchgate",
	Usage:     "compare two benchmark result files and report packages that regressed",
	ArgsUsage: "<base.txt> <pr.txt>",
	Action:    runBenchgate,
}

// benchRegressThreshold は退化とみなす sec/op geomean 比。2 は2倍。
const benchRegressThreshold = 2.0

func runBenchgate(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return fmt.Errorf("usage: benchgate <base.txt> <pr.txt>")
	}
	return benchGate(cmd.Writer, cmd.Args().Get(0), cmd.Args().Get(1), benchRegressThreshold)
}

// benchKey はパッケージとベンチ名の組。
type benchKey struct{ pkg, name string }

// benchGate は base と pr のベンチファイルを読み、threshold 以上悪化したパッケージを out へ1行ずつ書く。
// 悪化が無ければ何も書かない。
func benchGate(out io.Writer, basePath, prPath string, threshold float64) error {
	base, err := loadBench(basePath)
	if err != nil {
		return fmt.Errorf("read base: %w", err)
	}
	pr, err := loadBench(prPath)
	if err != nil {
		return fmt.Errorf("read pr: %w", err)
	}
	for _, line := range benchRegressions(base, pr, threshold) {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

// loadBench はベンチファイルを読み、ベンチ名ごとの1操作あたり時間のサンプル列を返す。単位は ns/op でも
// sec/op でもよい。比を取るので単位は打ち消え、base と PR が同じ単位でありさえすればよい。設定行や
// 構文エラー行はベンチ結果でないので読み飛ばす。
func loadBench(path string) (map[benchKey][]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	out := map[benchKey][]float64{}
	r := benchfmt.NewReader(f, path)
	for r.Scan() {
		res, ok := r.Result().(*benchfmt.Result)
		if !ok {
			continue
		}
		v, ok := res.Value("sec/op")
		if !ok {
			if v, ok = res.Value("ns/op"); !ok {
				continue
			}
		}
		k := benchKey{res.GetConfig("pkg"), res.Name.String()}
		out[k] = append(out[k], v)
	}
	return out, r.Err()
}

// benchRegressions は両方に在るベンチをパッケージごとに突き合わせ、sec/op の median 比の geomean が
// threshold 以上のパッケージを "- <pkg>: geomean x<r>" の行にして名前順で返す。両方に在るベンチだけを
// 見るので、PR だけの新規ベンチや base だけの削除済みベンチには影響されない。
func benchRegressions(base, pr map[benchKey][]float64, threshold float64) []string {
	type acc struct {
		sumLog float64
		n      int
	}
	byPkg := map[string]*acc{}
	for k, bvals := range base {
		pvals, ok := pr[k]
		if !ok || len(bvals) == 0 || len(pvals) == 0 {
			continue
		}
		bm, pm := benchMedian(bvals), benchMedian(pvals)
		if bm <= 0 || pm <= 0 {
			continue
		}
		a := byPkg[k.pkg]
		if a == nil {
			a = &acc{}
			byPkg[k.pkg] = a
		}
		a.sumLog += math.Log(pm / bm)
		a.n++
	}

	var lines []string
	for p, a := range byPkg {
		geomean := math.Exp(a.sumLog / float64(a.n))
		if geomean >= threshold {
			lines = append(lines, fmt.Sprintf("- %s: geomean x%.2f", p, geomean))
		}
	}
	sort.Strings(lines)
	return lines
}

// benchMedian はサンプル列の中央値を返す。呼び出し側のスライスは変更しない。
func benchMedian(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
