package valueobjects

import "errors"

// UserID enforces basic identity constraints at the domain boundary.
type UserID string

func NewUserID(v string) (UserID, error) {
	if v == "" {
		return "", errors.New("user id is required")
	}
	return UserID(v), nil
}
