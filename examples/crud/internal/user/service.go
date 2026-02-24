package user

import (
	"fmt"
	"sync"
)

// Service manages user data in memory.
// Replace with a database repository in a real project.
type Service struct {
	mu      sync.RWMutex
	users   map[int64]*User
	counter int64
}

func NewService() *Service {
	svc := &Service{users: make(map[int64]*User)}

	// Seed data — the API returns something useful right away.
	svc.Create(CreateUserDTO{Name: "Alice", Email: "alice@example.com"})
	svc.Create(CreateUserDTO{Name: "Bob", Email: "bob@example.com"})

	return svc
}

func (s *Service) FindAll() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
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

func (s *Service) Create(dto CreateUserDTO) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	u := &User{
		ID:    s.counter,
		Name:  dto.Name,
		Email: dto.Email,
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
