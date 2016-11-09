package migration

import (
	"sort"

	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
	"github.com/Sirupsen/logrus"
)

type migrationStep interface {
	Name()	    string
	// All migrations coming from a lower version will be run.  If any migration for a step fails, other
	// migrations will not be run and the version will not be upgraded.
	MigrateTo() int
	// If there are multiple migrations with the same MigrateTo() version, this can be used to sort those migrations
	Order()     int
	// Applies the migration.  Returns any errors (or nil).  An error will abort remaining migrations for the same
	// MigrateTo() version.
	Apply(facade *facade.Facade, ctx datastore.Context) error
}

var (
	log              = logging.PackageLogger()

	migrations = map[int][]migrationStep{}
	lastMigration = 0
)

// Sort migrations by Order()
type migrationOrder []migrationStep
func (m migrationOrder) Len() int {
	return len(m)
}
func (m migrationOrder) Less(i, j int) bool {
	return m[i].Order() < m[j].Order()
}
func (m migrationOrder) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

// Migrations will add themselves to the list of available migrations.
func addMigration(migration migrationStep) {
	log.WithFields(logrus.Fields{
		"migration": migration.Name(),
	}).Info("Adding migration")
	// Store this into the migration list.
	version := migration.MigrateTo()
	step, found := migrations[version]
	if !found {
		step = []migrationStep{}
	}
	step = append(step, migration)
	migrations[version] = step

	// Save the latest migration version we have.
	if migration.MigrateTo() > lastMigration {
		lastMigration = migration.MigrateTo()
	}
}

// Causes all applicable migrations to be applied.  The new version returned will be the last successful
// migration applied.  For example, all version 1 migrations are run against a version 0 version, resulting
// in a new version of 1.  If there are any version 2 migrations, all version 2 migrations will then be run
// resulting in a new version of 2.  Order() is used to order migrations for the same MigrateTo() version.
func Migrate(facade *facade.Facade, ctx datastore.Context, version int) (int, error) {
	curVersion := version

	log.WithFields(logrus.Fields{
		"fromversion": version,
		"toversion": lastMigration,
	}).Debug("Trying to perform migration")

	for i := version + 1; i <= lastMigration; i++ {
		// Anything within this loop is a migration; log it with Info
		log.WithField("version", curVersion).Info("Applying migrations")
		if step, found := migrations[i]; found {
			log.Infof("Found %d migration(s) to bring version up to %d\n", len(step), i)
			// Sort the migration step by Order()
			sort.Sort(migrationOrder(step))
			// Apply each migration in the step
			var migration migrationStep
			for _, migration = range step {
				log.WithField("migration", migration.Name()).Info("Applying migration")
				if err := migration.Apply(facade, ctx); err != nil {
					// Return the last successful migration version.
					log.WithField("migration", migration.Name()).WithError(err).
						Info("Migration error")
					return curVersion, err
				}
			}
		}
		log.WithField("toversion", i).Info("Completed migrations")
		curVersion = i
	}
	if version != curVersion {
		log.Infof("Completing migration; new version will be %d\n", curVersion)
	}
	return curVersion, nil
}
