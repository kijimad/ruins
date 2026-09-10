package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/designdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildImageTableFrom_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result, err := buildImageTableFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "*no images*", result)
}

func TestBuildImageTableFrom_NonExistentDir(t *testing.T) {
	t.Parallel()
	_, err := buildImageTableFrom("/nonexistent/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
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

func TestGenReadme_成功時にプレースホルダを置換したREADMEを生成する(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	tmplPath := filepath.Join(base, "README.tmpl.md")
	outPath := filepath.Join(base, "README.md")
	imgDir := filepath.Join(base, "img")
	designDir := filepath.Join(base, "design")

	tmpl := "# タイトル\n\n<!-- VRT_IMAGES -->\n\n## 状況\n\n<!-- DESIGN_STATUS -->\n"
	require.NoError(t, os.WriteFile(tmplPath, []byte(tmpl), 0o644))
	require.NoError(t, os.MkdirAll(imgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(imgDir, "TestGolden_Foo.png"), []byte("dummy"), 0o644))
	require.NoError(t, os.MkdirAll(designDir, 0o755))
	writeDoc(t, designDir, "a.md", "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# A\n")

	var buf bytes.Buffer
	require.NoError(t, genReadme(&buf, tmplPath, outPath, imgDir, designDir))
	assert.Equal(t, "Generated "+outPath+" from "+tmplPath+" ("+imgDir+")\n", buf.String())

	table, err := buildImageTableFrom(imgDir)
	require.NoError(t, err)
	docs, err := designdoc.LoadDir(designDir)
	require.NoError(t, err)
	statusTable := designdoc.RenderStatusSection(docs)
	want := strings.Replace(tmpl, placeholder, table, 1)
	want = strings.Replace(want, designStatusPlacehldr, statusTable, 1)

	got, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

func TestGenReadme_テンプレートが無ければエラーを返す(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	err := genReadme(io.Discard, filepath.Join(base, "missing.tmpl.md"),
		filepath.Join(base, "README.md"), filepath.Join(base, "img"), filepath.Join(base, "design"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenReadme_画像テーブル構築に失敗すればエラーを返す(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	tmplPath := filepath.Join(base, "README.tmpl.md")
	require.NoError(t, os.WriteFile(tmplPath, []byte("<!-- VRT_IMAGES -->"), 0o644))

	err := genReadme(io.Discard, tmplPath, filepath.Join(base, "README.md"),
		filepath.Join(base, "nonexistent-img"), filepath.Join(base, "design"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenReadme_設計ドキュメント読み込みに失敗すればエラーを返す(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	tmplPath := filepath.Join(base, "README.tmpl.md")
	require.NoError(t, os.WriteFile(tmplPath, []byte("<!-- VRT_IMAGES -->"), 0o644))
	imgDir := filepath.Join(base, "img")
	require.NoError(t, os.MkdirAll(imgDir, 0o755))

	err := genReadme(io.Discard, tmplPath, filepath.Join(base, "README.md"),
		imgDir, filepath.Join(base, "nonexistent-design"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
