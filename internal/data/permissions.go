package data

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/lib/pq"
)

// this holds all the perms for a user
type Permissions []string

// check if the user contains a specific perm by the perm's code
func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

// define model type
type PermissionsModel struct {
	DB *sql.DB
}

// func to get all perm codes for a specific user
// BEK Note: 16.3 again here i think this naming pattern is a bit odd, like why not GetAllPerms or GetUserPerms
func (m PermissionsModel) GetAllForUser(userID int64) (Permissions, error) {
	// pasted for correctness
	query := `
        SELECT permissions.code
        FROM permissions
        INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
        INNER JOIN users ON users_permissions.user_id = users.id
        WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// execute query
	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// declare a perms var to write perm values to
	var permissions Permissions

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}
	// ensure no error when iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// add provided perm codes for a user
func (m PermissionsModel) AddForUser(userID int64, codes ...string) error {
	// pasted for safety
	query := `
        INSERT INTO users_permissions
        SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID, pq.Array(codes))
	return err
}
