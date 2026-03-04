package user

import (
	"fmt"
	"sync"
)

// User is the internal user entity (includes password).
type User struct {
	ID       int64
	Name     string
	Email    string
	Password string // plaintext for demo — use bcrypt in production
}

// PublicUser is the API response entity (no password).
type PublicUser struct {
	ID    int64
	Name  string
	Email string
}

// Public returns a safe-to-serialize view of the user.
func (u *User) Public() PublicUser {
	return PublicUser{ID: u.ID, Name: u.Name, Email: u.Email}
}

// CreateUserDTO is used by both auth registration and direct user creation.
type CreateUserDTO struct {
	Name     string
	Email    string
	Password string
}

// UpdateUserDTO is the request body for PUT /users/{id}.
type UpdateUserDTO struct {
	Name  *string `validate:"max=50"`
	Email *string `validate:"email"`
}

// Service manages users in memory.
type Service struct {
	mu      sync.RWMutex
	users   map[int64]*User
	counter int64
}

func NewService() *Service {
	return &Service{users: make(map[int64]*User)}
}

func (s *Service) FindAll() []PublicUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]PublicUser, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u.Public())
	}
	return list
}

func (s *Service) FindOne(id int64) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return u, nil
}

func (s *Service) FindByEmail(email string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Email == email {
			return u
		}
	}
	return nil
}

func (s *Service) Create(dto CreateUserDTO) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	u := &User{
		ID:       s.counter,
		Name:     dto.Name,
		Email:    dto.Email,
		Password: dto.Password,
	}
	s.users[u.ID] = u
	return u
}

func (s *Service) Update(id int64, dto UpdateUserDTO) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}
	if dto.Name != nil {
		u.Name = *dto.Name
	}
	if dto.Email != nil {
		u.Email = *dto.Email
	}
	return u, nil
}

func (s *Service) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return fmt.Errorf("user %d not found", id)
	}
	delete(s.users, id)
	return nil
}
