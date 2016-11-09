package migration

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/logging"
)

func init() {
	// Add this sample to the list of migrations.  Because it has a MigrateTo() = 1, any uninitialized
	// CC or version 0 CC will run this migration, bringing the new migration version to 1.
	addMigration(migrateSample{})
}

// The migration sample doesn't do anything.  Adding it as a migration in init() will cause a
// new or upgraded CC to migrate to version 1 (MigrateTo()); but it doesn't perform any actions in Apply()
type migrateSample struct {}

func (m migrateSample) Name() string {
	return "Sample Migration: version to 1"
}

func (m migrateSample) MigrateTo() int {
	return 1
}

func (m migrateSample) Order() int {
	return 0
}

func (m migrateSample) Apply(f *facade.Facade, ctx datastore.Context) error {
	log := logging.PackageLogger().WithField("migration", "migrateSample")
	log.Info("Migration sample Apply()")

	// TODO: The migration would perform some actions, returning nil if there are no problems.

	return nil
}
