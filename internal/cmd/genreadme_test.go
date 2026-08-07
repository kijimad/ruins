package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildImageTableFrom_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result, err := buildImageTableFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "*画像なし*", result)
}

func TestBuildImageTableFrom_NonExistentDir(t *testing.T) {
	t.Parallel()
	_, err := buildImageTableFrom("/nonexistent/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "読み込みに失敗")
}

func TestBuildImageTableFrom_SingleImage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "TestGolden_Menu.png"), []byte("dummy"), 0644))

	result, err := buildImageTableFrom(dir)
	require.NoError(t, err)

	want := strings.ReplaceAll(`| | | | |
|---|---|---|---|
| <img src="DIR/TestGolden_Menu.png" width="200" /><br>Menu | | | |
`, "DIR", dir)
	assert.Equal(t, want, result)
}

func TestBuildImageTableFrom_MultipleImages(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	names := []string{
		"TestGolden_Alpha.png",
		"TestGolden_Beta.png",
		"TestGolden_Gamma.png",
		"TestGolden_Delta.png",
		"TestGolden_Epsilon.png",
	}
	for _, n := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte("dummy"), 0644))
	}

	result, err := buildImageTableFrom(dir)
	require.NoError(t, err)

	// ソート順: Alpha, Beta, Delta, Epsilon, Gamma
	want := strings.ReplaceAll(`| | | | |
|---|---|---|---|
| <img src="DIR/TestGolden_Alpha.png" width="200" /><br>Alpha | <img src="DIR/TestGolden_Beta.png" width="200" /><br>Beta | <img src="DIR/TestGolden_Delta.png" width="200" /><br>Delta | <img src="DIR/TestGolden_Epsilon.png" width="200" /><br>Epsilon |
| <img src="DIR/TestGolden_Gamma.png" width="200" /><br>Gamma | | | |
`, "DIR", dir)
	assert.Equal(t, want, result)
}

func TestBuildImageTableFrom_IgnoresNonPNG(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("text"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "image.jpg"), []byte("jpg"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "TestGolden_Only.png"), []byte("png"), 0644))

	result, err := buildImageTableFrom(dir)
	require.NoError(t, err)

	want := strings.ReplaceAll(`| | | | |
|---|---|---|---|
| <img src="DIR/TestGolden_Only.png" width="200" /><br>Only | | | |
`, "DIR", dir)
	assert.Equal(t, want, result)
}

func TestBuildImageTableFrom_IgnoresSubdirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir.png"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "TestGolden_Real.png"), []byte("png"), 0644))

	result, err := buildImageTableFrom(dir)
	require.NoError(t, err)

	want := strings.ReplaceAll(`| | | | |
|---|---|---|---|
| <img src="DIR/TestGolden_Real.png" width="200" /><br>Real | | | |
`, "DIR", dir)
	assert.Equal(t, want, result)
}

func TestBuildImageTableFrom_ExactColumns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, n := range []string{"A.png", "B.png", "C.png", "D.png"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte("dummy"), 0644))
	}

	result, err := buildImageTableFrom(dir)
	require.NoError(t, err)

	want := strings.ReplaceAll(`| | | | |
|---|---|---|---|
| <img src="DIR/A.png" width="200" /><br>A | <img src="DIR/B.png" width="200" /><br>B | <img src="DIR/C.png" width="200" /><br>C | <img src="DIR/D.png" width="200" /><br>D |
`, "DIR", dir)
	assert.Equal(t, want, result)
}
