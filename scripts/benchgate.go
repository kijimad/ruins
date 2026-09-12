//go:build ignore

// benchgate は base.txt と pr.txt のベンチ結果を読み、パッケージごとに sec/op の geomean 比を出して
// 2倍以上の悪化を検出する。両方に存在するベンチだけを突き合わせるので、PR だけにある新規ベンチや、
// 出力に紛れる go: 行に左右されない。悪化したパッケージを標準出力へ1行ずつ出し、無ければ何も出さない。
//
// 使い方: go run scripts/benchgate.go base.txt pr.txt
package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"golang.org/x/perf/benchfmt"
)

// key はパッケージとベンチ名の組。
type key struct{ pkg, name string }

// load はファイルからベンチ名ごとの1操作あたり時間のサンプル列を読む。単位は ns/op でも sec/op でも
// よい。比を取るので単位は打ち消え、base と PR が同じ単位でありさえすればよい。
func load(path string) (map[key][]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[key][]float64{}
	r := benchfmt.NewReader(f, path)
	for r.Scan() {
		res, ok := r.Result().(*benchfmt.Result)
		if !ok {
			continue // 設定行・空行・構文エラー行はベンチ結果でないので飛ばす
		}
		v, ok := res.Value("sec/op")
		if !ok {
			if v, ok = res.Value("ns/op"); !ok {
				continue
			}
		}
		k := key{res.GetConfig("pkg"), res.Name.String()}
		out[k] = append(out[k], v)
	}
	return out, r.Err()
}

func median(xs []float64) float64 {
	sort.Float64s(xs)
	n := len(xs)
	if n%2 == 1 {
		return xs[n/2]
	}
	return (xs[n/2-1] + xs[n/2]) / 2
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: benchgate base.txt pr.txt")
		os.Exit(2)
	}
	base, err := load(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "base:", err)
		os.Exit(2)
	}
	pr, err := load(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "pr:", err)
		os.Exit(2)
	}

	// パッケージごとに、両方に存在するベンチの pr/base 比の対数和を集める。geomean = exp(平均(log 比))。
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
		bm, pm := median(bvals), median(pvals)
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

	// 悪化パッケージを名前順に出す。2倍以上の geomean だけを退化とみなす。
	pkgs := make([]string, 0, len(byPkg))
	for p := range byPkg {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, p := range pkgs {
		a := byPkg[p]
		geomean := math.Exp(a.sumLog / float64(a.n))
		if geomean >= 2 {
			fmt.Printf("- %s: geomean x%.2f\n", p, geomean)
		}
	}
}
