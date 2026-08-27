package setup

import (
	"io/fs"
	"log/slog"
	"os"

	"github.com/spf13/viper"
	"github.com/user0608/bobi/configs"
	"github.com/user0608/bobi/connection"
	"github.com/user0608/bobi/httpserver"
	"github.com/user0608/bobi/jwtkeys"
	"github.com/user0608/bobi/setup/appenv"
	"github.com/user0608/bobi/setup/migrations"
	"github.com/user0608/bobi/setup/spa"
	"github.com/user0608/bobi/setup/web"
	"go.uber.org/fx"
)

type Service struct {
	version          string
	migrationFS      fs.FS
	spaFS            fs.FS // ReactJS
	webFS            fs.FS
	skipConfigLoad   bool
	skipDBConnection bool
}

func NewService(opts ...Option) *Service {
	s := &Service{}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Service) Run(opts ...fx.Option) {
	if s.migrationFS != nil {
		if action, ok := migrations.ParseMigrateCommand(os.Args[1:]); ok {
			s.runMigration(action)
			return
		}
	}

	var options = s.baseOptions()

	options = append(options, fx.Provide(s.jwtKeysConfig, jwtkeys.NewJwtKeyStore))
	options = append(options, opts...)
	options = append(options,
		fx.Provide(s.httpServerConfig),
		httpserver.Module,
	)

	if s.spaFS != nil && s.webFS == nil {
		options = append(options, fx.Provide(
			httpserver.AsRoute(func() *spa.SPAHandler {
				return spa.NewSPAHandler(s.spaFS, "/")
			}),
		))
	}

	if s.webFS != nil && s.spaFS == nil {
		options = append(options, fx.Provide(
			httpserver.AsRoute(func() *web.WebHandler {
				return web.NewWebHandler(s.webFS, "/")
			}),
		))
	}

	if s.spaFS != nil && s.webFS != nil {
		slog.Warn("both SPA and web filesystems are configured; their root routes may conflict")
	}

	options = append(options, fx.Invoke(httpserver.StartWebServer))

	fx.New(options...).Run()
}

func (s *Service) baseOptions() []fx.Option {
	var options = []fx.Option{
		appenv.Module,
	}
	if !s.skipConfigLoad {
		options = append(options, fx.Provide(configs.LoadConfigFromCLIArgs))
	}
	if !s.skipDBConnection {
		options = append(options,
			fx.Provide(s.databaseConfig, connection.NewConnection),
		)
	}
	if s.migrationFS != nil {
		options = append(options,
			fx.Supply(migrations.MigrationFS(s.migrationFS)),
			fx.Provide(migrations.NewMigrationRunner),
		)
	}

	return options
}

func (s *Service) httpServerConfig(v *viper.Viper) (httpserver.HttpApiConfig, error) {
	var config httpserver.HttpApiConfig
	if err := v.Unmarshal(&config); err != nil {
		return httpserver.HttpApiConfig{}, err
	}
	return config, nil
}

func (s *Service) databaseConfig(v *viper.Viper) (connection.DatabaseConfig, error) {
	var config connection.DatabaseConfig
	if err := v.UnmarshalKey("database", &config); err != nil {
		return connection.DatabaseConfig{}, err
	}
	return config, nil
}

func (s *Service) jwtKeysConfig(v *viper.Viper) (jwtkeys.JwtKeysConfig, error) {
	var config jwtkeys.JwtKeysConfig
	if err := v.UnmarshalKey("database", &config); err != nil {
		return jwtkeys.JwtKeysConfig{}, err
	}
	return config, nil
}
