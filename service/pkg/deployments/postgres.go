package deployments

import (
	"fmt"
	"os"
	"path"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const ProjectName = "service"

func RunMigrations(url string) error {

	wd, err := os.Getwd()

	if err != nil {
		return fmt.Errorf("failed to get current execution directory - %w", err)
	}

	for len(wd) != 1 && path.Base(wd) != ProjectName {
		wd = path.Dir(wd)
	}

	if len(wd) == 1 {
		return fmt.Errorf("project root folder '%s' not found", ProjectName)
	}

	migrationsPath := fmt.Sprintf("file://%v/migrations", wd)
	m, err := migrate.New(migrationsPath, url)

	if err != nil {
		return fmt.Errorf("failed to create Migrate - %w", err)
	}

	err = m.Up()

	if err != nil {
		return fmt.Errorf("failed to run migrations - %w", err)
	}

	return nil
}
