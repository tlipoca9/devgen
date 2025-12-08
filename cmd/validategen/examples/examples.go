package examples

import (
	"fmt"
	"strings"
)

// Status is a custom int type with validation.
type Status int

// Validate validates the Status value.
func (s Status) Validate() error {
	if s < 0 || s > 10 {
		return fmt.Errorf("status must be between 0 and 10")
	}
	return nil
}

// Address represents a user address.
type Address struct {
	Street string
	City   string
}

// Validate validates the Address fields.
func (a Address) Validate() error {
	if a.Street == "" {
		return fmt.Errorf("street is required")
	}
	if a.City == "" {
		return fmt.Errorf("city is required")
	}
	return nil
}

// User represents a user model.
// validategen:@validate
type User struct {
	// validategen:@required
	// validategen:@gt(0)
	ID int64

	// validategen:@required
	// validategen:@min(2)
	// validategen:@max(50)
	Name string

	// validategen:@required
	// validategen:@email
	Email string

	// validategen:@gte(0)
	// validategen:@lte(150)
	Age int

	// validategen:@required
	// validategen:@min(8)
	Password string

	// validategen:@oneof(admin, user, guest)
	Role string

	// validategen:@url
	Website string

	// validategen:@uuid
	UUID string

	// validategen:@ip
	IP string

	// validategen:@alphanum
	// validategen:@len(6)
	Code string

	// validategen:@method(Validate)
	Address Address

	// validategen:@method(Validate)
	// validategen:@required
	OptionalAddress *Address

	// validategen:@method(Validate)
	Status Status

	// validategen:@method(Validate)
	Addresses []Address

	// validategen:@method(Validate)
	AddressMap map[string]Address
}

// postValidate performs custom validation after field validation.
func (x User) postValidate(errs []string) error {
	if x.Role == "admin" && x.Age < 18 {
		errs = append(errs, "admin must be at least 18 years old")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// Config represents application config.
// validategen:@validate
type Config struct {
	// validategen:@required
	Host string

	// validategen:@required
	// validategen:@min(1)
	// validategen:@max(65535)
	Port int

	// validategen:@min(1)
	Tags []string

	// validategen:@oneof(debug, info, warn, error)
	LogLevel string
}

// NetworkConfig demonstrates IP validation annotations.
// validategen:@validate
type NetworkConfig struct {
	// validategen:@ipv4
	IPv4Address string

	// validategen:@ipv6
	IPv6Address string

	// validategen:@ip
	AnyIPAddress string

	// validategen:@duration
	Timeout string

	// validategen:@duration
	// validategen:@duration_min(1s)
	// validategen:@duration_max(1h)
	RetryInterval string

	// validategen:@duration_min(100ms)
	// validategen:@duration_max(30s)
	RequestTimeout string
}

// Product demonstrates numeric comparison annotations.
// validategen:@validate
type Product struct {
	// validategen:@required
	// validategen:@gt(0)
	ID int64

	// validategen:@required
	Name string

	// validategen:@gte(0)
	Price float64

	// validategen:@eq(1)
	Version int

	// validategen:@ne(0)
	Stock int

	// validategen:@lt(100)
	Discount float32

	// validategen:@lte(1000)
	Weight uint
}

// StringPatterns demonstrates string pattern annotations.
// validategen:@validate
type StringPatterns struct {
	// validategen:@alpha
	FirstName string

	// validategen:@alphanum
	Username string

	// validategen:@numeric
	PhoneNumber string

	// validategen:@contains(example)
	Email string

	// validategen:@excludes(admin)
	DisplayName string

	// validategen:@startswith(https://)
	SecureURL string

	// validategen:@endswith(.com)
	Domain string

	// validategen:@regex(^[A-Z]{2}-\d{4}$)
	ProductCode string
}

// AllNumericTypes demonstrates all supported numeric types.
// validategen:@validate
type AllNumericTypes struct {
	// validategen:@gt(0)
	Int int

	// validategen:@gt(0)
	Int8 int8

	// validategen:@gt(0)
	Int16 int16

	// validategen:@gt(0)
	Int32 int32

	// validategen:@gt(0)
	Int64 int64

	// validategen:@gt(0)
	Uint uint

	// validategen:@gt(0)
	Uint8 uint8

	// validategen:@gt(0)
	Uint16 uint16

	// validategen:@gt(0)
	Uint32 uint32

	// validategen:@gt(0)
	Uint64 uint64

	// validategen:@gt(0)
	Float32 float32

	// validategen:@gt(0)
	Float64 float64

	// validategen:@gt(0)
	Byte byte

	// validategen:@gt(0)
	Rune rune

	// validategen:@gt(0)
	Uintptr uintptr
}

// BoolExample demonstrates bool validation.
// validategen:@validate
type BoolExample struct {
	// validategen:@required
	IsActive bool

	// validategen:@eq(true)
	MustBeTrue bool
}

// SliceExample demonstrates slice validation.
// validategen:@validate
type SliceExample struct {
	// validategen:@required
	// validategen:@min(1)
	// validategen:@max(10)
	Items []string

	// validategen:@len(3)
	FixedItems []int
}

// FormatExample demonstrates format validation annotations.
// validategen:@validate
type FormatExample struct {
	// validategen:@format(json)
	JSONConfig string

	// validategen:@format(yaml)
	YAMLConfig string

	// validategen:@format(toml)
	TOMLConfig string

	// validategen:@format(csv)
	CSVData string
}
