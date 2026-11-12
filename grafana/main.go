package main

import (
	"context"
	"errors"
	"fmt"
	"grafana/internal/dashboards"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/urfave/cli/v3"
)

const (
	ExitOK          = 0
	ExitGeneral     = 1
	ExitNoInput     = 66
	ExitUnavailable = 69
	ExitSoftware    = 70
	ExitConfig      = 78
)

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

type generator struct {
	filename string
	build    func() ([]byte, error)
}

func main() {
	if err := cmdRun(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		if exitErr, ok := errors.AsType[*exitError](err); ok {
			os.Exit(exitErr.code)
		}

		os.Exit(ExitGeneral)
	}
}

func cmdRun() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cmd := &cli.Command{
		Name:                   "slotify-dashboards",
		Usage:                  "Generate Grafana Dashboard V2 resources",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output-dir",
				Aliases: []string{"o"},
				Usage:   "Path to output directory",
				Value:   "./dashboards",
				Sources: cli.EnvVars("OUTPUT_DIR"),
			},
			&cli.StringFlag{
				Name:    "folder-uid",
				Usage:   "Grafana folder UID",
				Sources: cli.EnvVars("GRAFANA_FOLDER_UID"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			outDir := cmd.String("output-dir")
			if outDir == "" {
				return &exitError{
					code: ExitConfig,
					err:  errors.New("output-dir cannot be empty"),
				}
			}

			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return &exitError{
					code: ExitSoftware,
					err:  fmt.Errorf("create output dir: %w", err),
				}
			}

			root, err := os.OpenRoot(outDir)
			if err != nil {
				return &exitError{
					code: ExitSoftware,
					err:  fmt.Errorf("open output root %s: %w", outDir, err),
				}
			}

			defer func() {
				if err := root.Close(); err != nil {
					fmt.Fprintf(
						os.Stderr,
						"close output root %s: %v\n",
						outDir,
						err,
					)
				}
			}()

			generators := []generator{
				{
					filename: "app-dashboard.json",
					build: func() ([]byte, error) {
						return dashboards.BuildAppDashboard()
					},
				},
			}

			for _, gen := range generators {
				raw, err := gen.build()
				if err != nil {
					return &exitError{
						code: ExitSoftware,
						err:  fmt.Errorf("%s: %w", gen.filename, err),
					}
				}

				if err := root.WriteFile(gen.filename, raw, 0o600); err != nil {
					return &exitError{
						code: ExitSoftware,
						err:  fmt.Errorf("write %s: %w", gen.filename, err),
					}
				}

				fmt.Printf(
					"generated %s\n",
					filepath.Join(outDir, gen.filename),
				)
			}

			return nil
		},
	}

	return cmd.Run(ctx, os.Args)
}
