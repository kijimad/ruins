package cmd

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
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

	t.Run("medianが0以下のベンチは集約から外し残りで判定する", func(t *testing.T) {
		t.Parallel()
		base := map[benchKey][]float64{k("a", "X"): {0, 0}, k("a", "Y"): {100}}
		pr := map[benchKey][]float64{k("a", "X"): {100}, k("a", "Y"): {400}}
		assert.Equal(t, []string{"- a: geomean x4.00"}, benchRegressions(base, pr, thr), "baseのmedianが0以下")

		base = map[benchKey][]float64{k("a", "X"): {100}, k("a", "Y"): {100}}
		pr = map[benchKey][]float64{k("a", "X"): {0, 0}, k("a", "Y"): {400}}
		assert.Equal(t, []string{"- a: geomean x4.00"}, benchRegressions(base, pr, thr), "prのmedianが0以下")
	})
}

func TestLoadBench_構文エラー行は読み飛ばす(t *testing.T) {
	t.Parallel()

	path := writeBenchFile(t, t.TempDir(), "bench.txt", "BenchmarkBroken-4 abc\nBenchmarkFoo-4 \t1000\t100 ns/op\n")

	got, err := loadBench(path)
	require.NoError(t, err)
	require.Len(t, got, 1, "構文エラー行はベンチ結果に数えない")
	vals, ok := got[benchKey{pkg: "example/x", name: "Foo-4"}]
	require.True(t, ok)
	require.Len(t, vals, 1)
	assert.InDelta(t, 1e-7, vals[0], 1e-12, "100 ns/op は sec/op に変換される")
}

func TestLoadBench_sec_opもns_opも無い測定値は読み飛ばす(t *testing.T) {
	t.Parallel()

	path := writeBenchFile(t, t.TempDir(), "bench.txt", "BenchmarkCustomOnly-4 \t1000\t42 custom/op\n")

	got, err := loadBench(path)
	require.NoError(t, err)
	assert.Empty(t, got, "時間の単位を持たない測定値は結果に数えない")
}

func TestBenchGate_ファイルを読んで比較する(t *testing.T) {
	t.Parallel()

	// go: 行や設定行が混ざっても benchfmt が結果行だけ拾う。PR だけの新規ベンチは判定に入らない。
	dir := t.TempDir()
	basePath := writeBenchFile(t, dir, "base.txt", "BenchmarkFoo-4 \t1000\t100 ns/op\nBenchmarkFoo-4 \t1000\t100 ns/op\n")
	prPath := writeBenchFile(t, dir, "pr.txt", "go: downloading example.com/mod v1.2.3\n"+
		"BenchmarkFoo-4 \t1000\t300 ns/op\nBenchmarkFoo-4 \t1000\t300 ns/op\n"+
		"BenchmarkNew-4 \t1000\t9999 ns/op\n")

	var buf bytes.Buffer
	require.NoError(t, benchGate(&buf, basePath, prPath, 2.0))
	assert.Equal(t, "- example/x: geomean x3.00\n", buf.String())
}

func TestBenchGate_閾値未満の悪化は報告しない(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	basePath := writeBenchFile(t, dir, "base.txt", "BenchmarkFoo-4 \t1000\t100 ns/op\n")
	prPath := writeBenchFile(t, dir, "pr.txt", "BenchmarkFoo-4 \t1000\t300 ns/op\n")

	// 3倍悪化だが、閾値を5倍にすると報告対象から外れる
	var buf bytes.Buffer
	require.NoError(t, benchGate(&buf, basePath, prPath, 5.0))
	assert.Empty(t, buf.String())
}

func TestBenchGate_読めないファイルはエラー(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var buf bytes.Buffer
	err := benchGate(&buf, filepath.Join(dir, "base.txt"), filepath.Join(dir, "pr.txt"), 2.0)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestBenchGate_prファイルが読めなければエラー(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	basePath := writeBenchFile(t, dir, "base.txt", "")

	var buf bytes.Buffer
	err := benchGate(&buf, basePath, filepath.Join(dir, "pr.txt"), 2.0)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

// writeBenchFile は共通の環境行の後ろに body を付けたベンチ結果のファイルを dir に書き、パスを返す
func writeBenchFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	const header = "goos: linux\ngoarch: amd64\npkg: example/x\ncpu: TestCPU\n"
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(header+body), 0o644))
	return path
}

// errBoom は failingWriter が返す書き込みのエラー。
var errBoom = errors.New("boom")

// failingWriter は常に書き込みに失敗するio.Writer。
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errBoom }

func TestBenchGate_出力に失敗すればエラーを返す(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	basePath := writeBenchFile(t, dir, "base.txt", "BenchmarkFoo-4 \t1000\t100 ns/op\n")
	prPath := writeBenchFile(t, dir, "pr.txt", "BenchmarkFoo-4 \t1000\t300 ns/op\n")

	err := benchGate(failingWriter{}, basePath, prPath, 2.0)
	require.ErrorIs(t, err, errBoom)
}

func benchgateApp() *cli.Command {
	return &cli.Command{
		Name:      "benchgate",
		ArgsUsage: "<base.txt> <pr.txt>",
		Action:    runBenchgate,
	}
}

func TestRunBenchgate_引数が2個でなければ使い方を返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"0個", nil},
		{"1個", []string{"base.txt"}},
		{"3個", []string{"a.txt", "b.txt", "c.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := benchgateApp().Run(context.Background(), append([]string{"benchgate"}, tt.args...))
			require.ErrorIs(t, err, errBenchgateUsage)
		})
	}
}

func TestRunBenchgate_2個の引数をbase_prの順でbenchGateへ渡す(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	basePath := writeBenchFile(t, dir, "base.txt", "BenchmarkFoo-4 \t1000\t100 ns/op\nBenchmarkFoo-4 \t1000\t100 ns/op\n")
	prPath := writeBenchFile(t, dir, "pr.txt", "BenchmarkFoo-4 \t1000\t300 ns/op\nBenchmarkFoo-4 \t1000\t300 ns/op\n")

	var buf bytes.Buffer
	app := benchgateApp()
	app.Writer = &buf

	err := app.Run(context.Background(), []string{"benchgate", basePath, prPath})
	require.NoError(t, err)
	assert.Equal(t, "- example/x: geomean x3.00\n", buf.String(), "先頭引数をbase、2番目をprとして渡す")
}
