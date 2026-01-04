package collector

import (
	"context"
	"database/sql"
	"fmt"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLRoles 采集所有 PostgreSQL 角色信息
func ScrapePostgreSQLRoles(db *sql.DB) (*model.PostgreSQLRoleStats, error) {
	ctx := context.Background()

	stats := &model.PostgreSQLRoleStats{
		Roles: []*model.PostgreSQLRole{},
	}

	rows, err := db.QueryContext(ctx, `
        SELECT 
            r.rolname,
            r.rolsuper,
            r.rolcreatedb,
            r.rolcreaterole,
            r.rolinherit,
            r.rolreplication,
            r.rolcanlogin,
            r.rolconnlimit,
            r.rolvaliduntil,
        FROM pg_roles r
        WHERE r.rolname NOT IN ('pg_signal_backend', 'rds_iam', 'rds_replication', 'rds_superuser');
    `)
	if err != nil {
		return stats, fmt.Errorf("failed to query roles: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name sql.NullString
		var superuser sql.NullBool
		var createDB sql.NullBool
		var createUser sql.NullBool
		var inherit sql.NullBool
		var replication sql.NullBool
		var canLogin sql.NullBool
		var connLimit sql.NullInt64
		var validUntil sql.NullTime

		err := rows.Scan(
			&name,
			&superuser,
			&createDB,
			&createUser,
			&inherit,
			&replication,
			&canLogin,
			&connLimit,
			&validUntil,
		)
		if err != nil {
			continue
		}

		isDefault := false
		if name.String == "postgres" || name.String == "public" {
			isDefault = true
		}

		stats.Roles = append(stats.Roles, &model.PostgreSQLRole{
			Name:            coalesceNullString(name),
			Superuser:       superuser.Valid && superuser.Bool,
			CreateDB:        createDB.Valid && createDB.Bool,
			CreateUser:      createUser.Valid && createUser.Bool,
			Inherit:         inherit.Valid && inherit.Bool,
			Replication:     replication.Valid && replication.Bool,
			CanLogin:        canLogin.Valid && canLogin.Bool,
			ConnectionLimit: coalesceNullInt64(connLimit),
			ValidUntil:      &validUntil.Time,
			IsDefault:       isDefault,
		})
	}

	return stats, nil
}
