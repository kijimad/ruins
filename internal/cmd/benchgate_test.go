package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchMedian(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 2.0, benchMedian([]float64{3, 1, 2}), 1e-9, "奇数個は中央")
	assert.InDelta(t, 2.5, benchMedian([]float64{4, 2, 1, 3}), 1e-9, "偶数個は中央2つの平均")

	in := []float64{3, 1, 2}
	_ = benchMedian(in)
	assert.Equal(t, []float64{3, 1, 2}, in, "呼び出し側のスライスを変更しない")
}

func TestBenchRegressions(t *testing.T) {
	t.Parallel()

	const thr = 2.0
	k := func(pkg, name string) benchKey { return benchKey{pkg: pkg, name: name} }

	t.Run("悪化なしは空", func(t *testing.T) {
		t.Parallel()
		base := map[benchKey][]float64{k("a", "X"): {100, 100}}
		pr := map[benchKey][]float64{k("a", "X"): {101, 101}}
		assert.Empty(t, benchRegressions(base, pr, thr))
	})

	t.Run("3倍悪化は名前付きで報告", func(t *testing.T) {
		t.Parallel()
		base := map[benchKey][]float64{k("a", "X"): {100}}
		pr := map[benchKey][]float64{k("a", "X"): {300}}
		assert.Equal(t, []string{"- a: geomean x3.00"}, benchRegressions(base, pr, thr))
	})

	t.Run("ちょうど2倍はしきい値に含む", func(t *testing.T) {
		t.Parallel()
		base := map[benchKey][]float64{k("a", "X"): {100}}
		pr := map[benchKey][]float64{k("a", "X"): {200}}
		assert.Equal(t, []string{"- a: geomean x2.00"}, benchRegressions(base, pr, thr))
	})

	t.Run("PRだけの新規ベンチは無視", func(t *testing.T) {
		t.Parallel()
		base := map[benchKey][]float64{k("a", "X"): {100}}
		pr := map[benchKey][]float64{k("a", "X"): {100}, k("a", "New"): {999999}}
		assert.Empty(t, benchRegressions(base, pr, thr), "base に無い新規ベンチは巨大でも判定に入れない")
	})

	t.Run("baseだけの削除済みベンチは無視", func(t *testing.T) {
		t.Parallel()
		base := map[benchKey][]float64{k("a", "X"): {100}, k("a", "Gone"): {100}}
		pr := map[benchKey][]float64{k("a", "X"): {100}}
		assert.Empty(t, benchRegressions(base, pr, thr))
	})

	t.Run("パッケージ単位で独立に判定", func(t *testing.T) {
		t.Parallel()
		base := map[benchKey][]float64{k("a", "X"): {100}, k("b", "Y"): {100}}
		pr := map[benchKey][]float64{k("a", "X"): {300}, k("b", "Y"): {101}}
		assert.Equal(t, []string{"- a: geomean x3.00"}, benchRegressions(base, pr, thr), "悪化した a だけ。b は無視")
	})

	t.Run("パッケージ内は比の geomean で集約", func(t *testing.T) {
		t.Parallel()
		// x4 と x1 の geomean は 2 なのでちょうどしきい値
		base := map[benchKey][]float64{k("a", "X"): {100}, k("a", "Y"): {100}}
		pr := map[benchKey][]float64{k("a", "X"): {400}, k("a", "Y"): {100}}
		assert.Equal(t, []string{"- a: geomean x2.00"}, benchRegressions(base, pr, thr))
	})
}

func TestBenchGate_ファイルを読んで比較する(t *testing.T) {
	t.Parallel()

	// go: 行や設定行が混ざっても benchfmt が結果行だけ拾う。PR だけの新規ベンチは判定に入らない。
	header := "goos: linux\ngoarch: amd64\npkg: example/x\ncpu: TestCPU\n"
	baseBody := header + "BenchmarkFoo-4 \t1000\t100 ns/op\nBenchmarkFoo-4 \t1000\t100 ns/op\n"
	prBody := header + "go: downloading example.com/mod v1.2.3\n" +
		"BenchmarkFoo-4 \t1000\t300 ns/op\nBenchmarkFoo-4 \t1000\t300 ns/op\n" +
		"BenchmarkNew-4 \t1000\t9999 ns/op\n"

	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.txt")
	prPath := filepath.Join(dir, "pr.txt")
	require.NoError(t, os.WriteFile(basePath, []byte(baseBody), 0o644))
	require.NoError(t, os.WriteFile(prPath, []byte(prBody), 0o644))

	var buf bytes.Buffer
	require.NoError(t, benchGate(&buf, basePath, prPath, 2.0))
	assert.Equal(t, "- example/x: geomean x3.00\n", buf.String())
}

func TestBenchGate_読めないファイルはエラー(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := benchGate(&buf, filepath.Join(t.TempDir(), "missing.txt"), filepath.Join(t.TempDir(), "missing.txt"), 2.0)
	require.Error(t, err)
}
