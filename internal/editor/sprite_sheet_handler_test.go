package editor

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleSpriteSheets(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddSpriteSheet(raw.SpriteSheet{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sprite-sheets", nil)
	srv.handleSpriteSheets(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleSpriteSheetCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新スプライト"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/sprite-sheets/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleSpriteSheetCreate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	sheets := srv.store.SpriteSheets()
	require.Len(t, sheets, 1)
	assert.Equal(t, "新スプライト", sheets[0].Name)
}

func TestHandleCutter(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/cutter", nil)
	srv.handleCutter(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Sprite Cutter")
}

func TestHandleCutterUpload(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	pngData := createTestPNG(t, 256, 256)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("sheet", "test.png")
	require.NoError(t, err)
	_, err = io.Copy(part, bytes.NewReader(pngData))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/cutter/upload", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	srv.handleCutterUpload(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, srv.uploadedSheet)
	assert.Equal(t, "/cutter", w.Header().Get("HX-Redirect"))
}

func TestHandleCutterSave(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	store := setupTestStore(t, []raw.Item{})
	srv := NewServer(store, WithOutputDir(outDir))

	// テスト画像をセットする
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := range 32 {
		for x := range 32 {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	srv.uploadedSheet = img

	form := url.Values{
		"name_0": {"test_sprite"},
		"name_1": {""},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/cutter/save", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleCutterSave(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "1 個のスプライトを保存しました")

	// ファイルが作成されたか確認する
	_, err := os.Stat(filepath.Join(outDir, "test_sprite_.png"))
	assert.NoError(t, err)
}

func TestHandleCutterSave_NoImage(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/cutter/save", nil)
	srv.handleCutterSave(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCutterSave_SkipsTransparent(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	store := setupTestStore(t, []raw.Item{})
	srv := NewServer(store, WithOutputDir(outDir))

	// 全セル透明な画像
	srv.uploadedSheet = image.NewRGBA(image.Rect(0, 0, 64, 64))

	form := url.Values{
		"name_0": {"transparent"},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/cutter/save", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleCutterSave(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "0 個のスプライトを保存しました")

	// ファイルが作成されていないことを確認する
	_, err := os.Stat(filepath.Join(outDir, "transparent_.png"))
	assert.True(t, os.IsNotExist(err))
}

func TestIsTransparent(t *testing.T) {
	t.Parallel()

	transparent := image.NewRGBA(image.Rect(0, 0, 32, 32))
	assert.True(t, isTransparent(transparent))

	opaque := image.NewRGBA(image.Rect(0, 0, 32, 32))
	opaque.Set(0, 0, color.RGBA{R: 255, A: 255})
	assert.False(t, isTransparent(opaque))
}

func TestClampUint8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   int
		want uint8
	}{
		{"normal", 128, 128},
		{"zero", 0, 0},
		{"max", 255, 255},
		{"negative", -1, 0},
		{"overflow", 300, 255},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, clampUint8(tt.in))
		})
	}
}
