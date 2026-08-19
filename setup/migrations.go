package setup

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/user0608/bobi/setup/migrations"
	"go.uber.org/fx"
)

func (s *Service) runMigration(action string) {
	if action == "script" {
		s.runMigrationScript()
		return
	}

	app := fx.New(
		fx.NopLogger,
		fx.Options(s.baseOptions()...),
		fx.Invoke(func(mr *migrations.MigrationRunner, shutdowner fx.Shutdowner) {
			exitCode := runMigrationAction(context.Background(), mr, action, os.Stdout)

			_ = shutdowner.Shutdown(fx.ExitCode(exitCode))
		}),
	)

	app.Run()
}

func (s *Service) runMigrationScript() {
	app := fx.New(
		fx.NopLogger,
		fx.Supply(migrations.MigrationFS(s.migrationFS)),
		fx.Provide(migrations.NewMigrationScriptRunner),
		fx.Invoke(func(mr *migrations.MigrationRunner, shutdowner fx.Shutdowner) {
			exitCode := runMigrationAction(context.Background(), mr, "script", os.Stdout)
			_ = shutdowner.Shutdown(fx.ExitCode(exitCode))
		}),
	)

	app.Run()
}

func runMigrationAction(ctx context.Context, mr *migrations.MigrationRunner, action string, out io.Writer) int {
	if err := executeMigrationAction(ctx, mr, action, out); err != nil {
		_, _ = fmt.Fprintln(out, err)
		return 1
	}

	return 0
}

func executeMigrationAction(ctx context.Context, mr *migrations.MigrationRunner, action string, out io.Writer) error {
	switch action {
	case "up":
		return mr.Up(ctx)
	case "down":
		return mr.Down(ctx)
	case "status":
		return mr.Status(ctx)
	case "script":
		str, err := mr.SQLScript()
		if err != nil {
			return err
		}

		_, err = io.WriteString(out, str)
		return err
	default:
		return fmt.Errorf("unknown migration action: %s", action)
	}
}
