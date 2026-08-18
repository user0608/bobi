package setup

import (
	"io/fs"
)

type Option func(*Service)

func WithVersion(version string) Option {
	return func(s *Service) {
		s.version = version
	}
}

func WithMigration(_fs fs.FS) Option {
	return func(s *Service) {
		if _fs != nil {
			s.migrationFS = _fs
		}
	}
}

func WithSkipConfig() Option {
	return func(s *Service) {
		s.skipConfigLoad = true
	}
}

func WithSkipDBConnection() Option {
	return func(s *Service) {
		s.skipDBConnection = true
	}
}
