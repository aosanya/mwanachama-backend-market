package market

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type uniqueKey struct {
	role    string
	columns []string
}

var uniqueKeys = []uniqueKey{
	{RoleLine, []string{"group_order_id", "picker"}},
	{RoleQuote, []string{"notice_id", "poster_id"}},
}

func Provision(db *gorm.DB, s *spec.Spec) error {
	if err := dropDuplicates(db, s); err != nil {
		return err
	}
	return spec.Migrate(db, s)
}

func dropDuplicates(db *gorm.DB, s *spec.Spec) error {
	for _, key := range uniqueKeys {
		for _, o := range s.Objects {
			if o.Role != key.role {
				continue
			}
			table := s.TableFor(o)
			if !db.Migrator().HasTable(table) {
				continue
			}
			stmt := fmt.Sprintf(
				"delete from %[1]s where id not in (select min(id) from %[1]s group by %[2]s)", table, strings.Join(key.columns, ", "))
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("market: drop duplicate %s rows: %w", key.role, err)
			}
		}
	}
	return nil
}
