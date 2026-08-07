package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestNewMainApp_サブコマンドが揃っている(t *testing.T) {
	t.Parallel()

	app := NewMainApp()

	assert.Equal(t, "ruins", app.Name)
	assert.Equal(t, consts.AppVersion, app.Version)

	names := make([]string, 0, len(app.Commands))
	for _, c := range app.Commands {
		names = append(names, c.Name)
	}
	assert.Equal(t, []string{"play", "simulate-balance", "genreadme", "gencomponents", "designdoc"}, names)
}

func TestRunMainApp_成功時はnilを返す(t *testing.T) {
	t.Parallel()

	app := &cli.Command{
		Name: "test",
		Action: func(_ context.Context, _ *cli.Command) error {
			return nil
		},
	}

	err := RunMainApp(app, "test")
	assert.NoError(t, err)
}

func TestRunMainApp_失敗時はエラーをラップする(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	app := &cli.Command{
		Name: "test",
		Action: func(_ context.Context, _ *cli.Command) error {
			return wantErr
		},
	}

	err := RunMainApp(app, "test")
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	assert.ErrorContains(t, err, "コマンド実行が失敗した")
}
