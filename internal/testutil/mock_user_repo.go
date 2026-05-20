package testutil

import (
	"context"

	"github.com/gliedabrennung/messenger-core/internal/apperr"
	"github.com/gliedabrennung/messenger-core/internal/entity"
)

type MockUserRepo struct {
	Users map[string]*entity.User
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{Users: make(map[string]*entity.User)}
}

func (m *MockUserRepo) Create(_ context.Context, user *entity.User) error {
	if _, ok := m.Users[user.Username]; ok {
		return apperr.ErrUserAlreadyExists
	}
	user.ID = int64(len(m.Users) + 1)
	m.Users[user.Username] = user
	return nil
}

func (m *MockUserRepo) GetByUsername(_ context.Context, username string) (*entity.User, error) {
	user, ok := m.Users[username]
	if !ok {
		return nil, apperr.ErrUserNotFound
	}
	return user, nil
}
