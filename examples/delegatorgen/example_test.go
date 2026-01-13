package delegatorgen

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel"
)

// TestUserRepositoryDelegator tests the generated delegator.
func TestUserRepositoryDelegator(t *testing.T) {
	// Create a base repository implementation
	baseRepo := &mockUserRepository{}

	// Create cache and tracer
	cache := NewInMemoryCache()
	tracer := otel.Tracer("test")

	// Build the delegator with cache and tracing
	repo := NewUserRepositoryDelegator(baseRepo).
		WithCache(cache).
		WithTracing(tracer).
		Build()

	ctx := context.Background()

	// First call - cache miss, calls base repository
	user, err := repo.GetByID(ctx, "user-123")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if user.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got '%s'", user.Name)
	}

	// Second call - cache hit, returns cached value
	user2, err := repo.GetByID(ctx, "user-123")
	if err != nil {
		t.Fatalf("GetByID (cached) failed: %v", err)
	}
	if user2.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got '%s'", user2.Name)
	}

	// Save invalidates cache
	user.Name = "Updated Name"
	if err := repo.Save(ctx, user); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Third call - should get updated value from base repo
	user3, err := repo.GetByID(ctx, "user-123")
	if err != nil {
		t.Fatalf("GetByID (after save) failed: %v", err)
	}
	if user3.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", user3.Name)
	}
}

// ExampleUserRepositoryDelegator demonstrates how to use the generated delegator.
func ExampleUserRepositoryDelegator() {
	// Create a base repository implementation
	baseRepo := &mockUserRepository{}

	// Create cache and tracer
	cache := NewInMemoryCache()
	tracer := otel.Tracer("example")

	// Build the delegator with cache and tracing
	repo := NewUserRepositoryDelegator(baseRepo).
		WithCache(cache).
		WithTracing(tracer).
		Build()

	// Use the repository - caching and tracing are automatic
	ctx := context.Background()

	// First call - cache miss, calls base repository
	user, err := repo.GetByID(ctx, "user-123")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got user: %s\n", user.Name)

	// Second call - cache hit, returns cached value
	user, err = repo.GetByID(ctx, "user-123")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got user (cached): %s\n", user.Name)

	// Save invalidates cache
	user.Name = "Updated Name"
	if err := repo.Save(ctx, user); err != nil {
		panic(err)
	}

	// Third call - cache miss again due to eviction
	user, err = repo.GetByID(ctx, "user-123")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got user (after save): %s\n", user.Name)
}

// mockUserRepository is a mock implementation for demonstration.
type mockUserRepository struct {
	users map[string]*User
}

func (m *mockUserRepository) GetByID(_ context.Context, id string) (*User, error) {
	if m.users == nil {
		m.users = make(map[string]*User)
		m.users["user-123"] = &User{ID: "user-123", Name: "John Doe", Email: "john@example.com"}
	}
	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	return user, nil
}

func (m *mockUserRepository) GetByEmail(_ context.Context, email string) (*User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found with email: %s", email)
}

func (m *mockUserRepository) List(_ context.Context, _, _ int) ([]*User, error) {
	var users []*User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *mockUserRepository) Save(_ context.Context, user *User) error {
	if m.users == nil {
		m.users = make(map[string]*User)
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(_ context.Context, id string) error {
	delete(m.users, id)
	return nil
}
