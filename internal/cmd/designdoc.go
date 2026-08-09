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

func runDesignDocValidate(_ context.Context, _ *cli.Command) error {
	docs, err := designdoc.LoadDir(designdoc.DefaultDir)
	if err != nil {
		return err
	}

	problems := designdoc.Validate(docs)
	for _, p := range problems {
		fmt.Printf("%s: %s\n", p.Path, p.Message)
	}

	if len(problems) > 0 {
		return errValidation
	}
	fmt.Printf("OK: validated %d documents\n", len(docs))

	return nil
}

func runDesignDocGen(_ context.Context, _ *cli.Command) error {
	changed, err := designdoc.BackfillDir(designdoc.DefaultDir)
	if err != nil {
		return err
	}

	for _, path := range changed {
		fmt.Printf("added: %s\n", path)
	}
	fmt.Printf("added frontmatter to %d documents\n", len(changed))

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
