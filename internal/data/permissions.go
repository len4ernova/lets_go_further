package data

import (
	"context"
	"database/sql"
	"slices"
	"time"
)

// лайс для хранения разрешений для одного user
type Permissions []string

// Include - метод проверяет по коду, имеется ли разрешении.
func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)

}

// PermissionModel
type PermissionModel struct {
	DB *sql.DB
}

// GetAllForUser - возвращает все коды разрешений для конкретного user
func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}
