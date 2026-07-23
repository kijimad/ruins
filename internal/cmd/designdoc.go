package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/kijimaD/ruins/internal/designdoc"
	"github.com/urfave/cli/v3"
)

// errValidation は frontmatter 検証で Error が見つかったことを表す。
var errValidation = errors.New("設計ドキュメントの検証に失敗した")

// CmdDesignDoc は設計ドキュメントの frontmatter を扱うサブコマンド
var CmdDesignDoc = &cli.Command{
	Name:        "designdoc",
	Usage:       "designdoc [validate|backfill|list]",
	Description: "docs/design の frontmatter を検証・付与・一覧する",
	Commands: []*cli.Command{
		{
			Name:   "validate",
			Usage:  "frontmatter の有無と妥当性、進捗との整合を検証する",
			Action: runDesignDocValidate,
		},
		{
			Name:   "backfill",
			Usage:  "frontmatter を欠くドキュメントに既定値を決定的に付与する",
			Action: runDesignDocBackfill,
		},
		{
			Name:  "list",
			Usage: "frontmatter で絞り込んで一覧する",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "status", Usage: "指定 status のみ"},
				&cli.StringFlag{Name: "auto", Usage: "指定 auto のみ"},
				&cli.StringFlag{Name: "tag", Usage: "指定タグを含むもののみ"},
				&cli.BoolFlag{Name: "open", Usage: "着手対象、すなわち open な status のみ"},
			},
			Action: runDesignDocList,
		},
	},
}

func runDesignDocValidate(_ context.Context, _ *cli.Command) error {
	docs, err := designdoc.LoadDir(designdoc.DefaultDir)
	if err != nil {
		return err
	}

	problems := designdoc.Validate(docs)
	for _, p := range problems {
		fmt.Printf("%-5s %s: %s\n", p.Severity, p.Path, p.Message)
	}

	if designdoc.HasError(problems) {
		return errValidation
	}
	fmt.Printf("OK: %d 件のドキュメントを検証した\n", len(docs))

	return nil
}

func runDesignDocBackfill(_ context.Context, _ *cli.Command) error {
	changed, err := designdoc.BackfillDir(designdoc.DefaultDir)
	if err != nil {
		return err
	}

	for _, path := range changed {
		fmt.Printf("付与: %s\n", path)
	}
	fmt.Printf("%d 件に frontmatter を付与した\n", len(changed))

	return nil
}

func runDesignDocList(_ context.Context, cmd *cli.Command) error {
	docs, err := designdoc.LoadDir(designdoc.DefaultDir)
	if err != nil {
		return err
	}

	status := cmd.String("status")
	auto := cmd.String("auto")
	tag := cmd.String("tag")
	openOnly := cmd.Bool("open")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tSTATUS\tAUTO\tPROGRESS\tTAGS")
	for _, doc := range docs {
		f := doc.Front
		if status != "" && string(f.Status) != status {
			continue
		}
		if auto != "" && string(f.Auto) != auto {
			continue
		}
		if tag != "" && !slices.Contains(f.Tags, tag) {
			continue
		}
		if openOnly && !f.Status.IsOpen() {
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
