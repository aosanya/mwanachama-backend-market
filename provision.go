package market

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func Provision(db *gorm.DB, s *spec.Spec) error {
	if err := dropDuplicateLines(db, s); err != nil {
		return err
	}
	return spec.Migrate(db, s)
}

func dropDuplicateLines(db *gorm.DB, s *spec.Spec) error {
	for _, o := range s.Objects {
		if o.Role != RoleLine {
			continue
		}
		table := s.TableFor(o)
		if !db.Migrator().HasTable(table) {
			return nil
		}
		stmt := fmt.Sprintf(
			"delete from %[1]s where id not in (select min(id) from %[1]s group by group_order_id, picker)", table)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("market: drop duplicate lines: %w", err)
		}
	}
	return nil
}
