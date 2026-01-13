package delegatorgen

import "context"

// User represents a user entity.
type User struct {
	ID    string
	Name  string
	Email string
	Age   int
}

// UserRepository is a repository interface for User operations.
// delegatorgen:@delegator
type UserRepository interface {
	// GetByID retrieves a user by ID.
	// Uses cache with 5 minute TTL and tracing.
	// delegatorgen:@cache(ttl=5m)
	// delegatorgen:@trace(attrs=id)
	GetByID(ctx context.Context, id string) (*User, error)

	// GetByEmail retrieves a user by email.
	// Uses cache with custom key template.
	// delegatorgen:@cache(ttl=10m, key="user:email:{email}")
	// delegatorgen:@trace
	GetByEmail(ctx context.Context, email string) (*User, error)

	// List retrieves users with pagination.
	// Uses cache with TTL jitter to prevent thundering herd.
	// delegatorgen:@cache(ttl=2m, jitter=30s, prefix="users:list")
	// delegatorgen:@trace(attrs=offset,limit)
	List(ctx context.Context, offset, limit int) ([]*User, error)

	// Save creates or updates a user.
	// Evicts cache entries for this user.
	// delegatorgen:@cache_evict(key="user:id:{user.ID}")
	// delegatorgen:@trace
	Save(ctx context.Context, user *User) error

	// Delete removes a user.
	// Evicts cache entries for this user.
	// delegatorgen:@cache_evict(key="user:id:{id}")
	// delegatorgen:@trace(attrs=id)
	Delete(ctx context.Context, id string) error
}

// OrderRepository is a repository interface for Order operations.
// delegatorgen:@delegator
type OrderRepository interface {
	// GetByID retrieves an order by ID with tracing only.
	// delegatorgen:@trace(name="order.get", attrs=id)
	GetByID(ctx context.Context, id string) (*Order, error)

	// Create creates a new order.
	// delegatorgen:@trace(name="order.create")
	Create(ctx context.Context, order *Order) error
}

// Order represents an order entity.
type Order struct {
	ID     string
	UserID string
	Amount float64
	Status string
}
