package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/kijimaD/ruins/internal/designdoc"
	"github.com/urfave/cli/v3"
)

// errValidation は frontmatter 検証で問題が見つかったことを表す。
var errValidation = errors.New("design document validation failed")

// CmdDesignDoc は設計ドキュメントの frontmatter を扱うサブコマンド
var CmdDesignDoc = &cli.Command{
	Name:        "designdoc",
	Usage:       "designdoc [gen|validate|list]",
	Description: "generate, validate, and list frontmatter under docs/design",
	Commands: []*cli.Command{
		{
			Name:   "gen",
			Usage:  "deterministically add default frontmatter to documents that lack it",
			Action: runDesignDocGen,
		},
		{
			Name:   "validate",
			Usage:  "validate frontmatter presence, validity, and consistency with progress",
			Action: runDesignDocValidate,
		},
		{
			Name:  "list",
			Usage: "list filtered by frontmatter; assumes validate has passed; --open treats invalid status as closed and excludes it",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "status", Usage: "only the specified status"},
				&cli.StringFlag{Name: "auto", Usage: "only the specified auto"},
				&cli.StringFlag{Name: "tag", Usage: "only documents containing the specified tag"},
				&cli.BoolFlag{Name: "open", Usage: "only actionable, that is open, status"},
			},
			Action: runDesignDocList,
		},
	},
}

// designDocFilter は list の絞り込み条件。空文字と false は無指定を表す。
type designDocFilter struct {
	status string
	auto   string
	tag    string
	open   bool
}

func runDesignDocValidate(_ context.Context, cmd *cli.Command) error {
	return designDocValidate(cmd.Writer, designdoc.DefaultDir)
}

// designDocValidate は dir 下の frontmatter を検証し、問題を out へ書く。問題があればエラーを返す。
func designDocValidate(out io.Writer, dir string) error {
	docs, err := designdoc.LoadDir(dir)
	if err != nil {
		return err
	}

	problems := designdoc.Validate(docs)
	for _, p := range problems {
		_, _ = fmt.Fprintf(out, "%s: %s\n", p.Path, p.Message)
	}

	if len(problems) > 0 {
		return errValidation
	}
	_, _ = fmt.Fprintf(out, "OK: validated %d documents\n", len(docs))

	return nil
}

func runDesignDocGen(_ context.Context, cmd *cli.Command) error {
	return designDocGen(cmd.Writer, designdoc.DefaultDir)
}

// designDocGen は dir 下の frontmatter を持たないドキュメントへ既定値を付与し、付与先を out へ書く。
func designDocGen(out io.Writer, dir string) error {
	changed, err := designdoc.BackfillDir(dir)
	if err != nil {
		return err
	}

	for _, path := range changed {
		_, _ = fmt.Fprintf(out, "added: %s\n", path)
	}
	_, _ = fmt.Fprintf(out, "added frontmatter to %d documents\n", len(changed))

	return nil
}

func runDesignDocList(_ context.Context, cmd *cli.Command) error {
	return designDocList(cmd.Writer, designdoc.DefaultDir, designDocFilter{
		status: cmd.String("status"),
		auto:   cmd.String("auto"),
		tag:    cmd.String("tag"),
		open:   cmd.Bool("open"),
	})
}

// designDocList は dir 下のドキュメントを filter で絞り、表として out へ書く。
func designDocList(out io.Writer, dir string, filter designDocFilter) error {
	docs, err := designdoc.LoadDir(dir)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tSTATUS\tAUTO\tPROGRESS\tTAGS")
	for _, doc := range docs {
		f := doc.Front
		if filter.status != "" && string(f.Status) != filter.status {
			continue
		}
		if filter.auto != "" && string(f.Auto) != filter.auto {
			continue
		}
		if filter.tag != "" && !slices.Contains(f.Tags, filter.tag) {
			continue
		}
		if filter.open && !f.Status.IsOpen() {
			continue
		}

		progress := "-"
		if doc.HasProgress {
			progress = fmt.Sprintf("%d/%d", doc.DoneTasks, doc.DoneTasks+doc.OpenTasks)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", doc.Path, f.Status, f.Auto, progress, strings.Join(f.Tags, ","))
	}

	return w.Flush()
}
