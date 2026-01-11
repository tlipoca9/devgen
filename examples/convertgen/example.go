package convertgen

// User is the source type.
type User struct {
	ID       int
	FullName string
	Email    string
	Password string
	Age      int
	Score    *int // pointer to basic type
	Address  *Address
	Tags     []string
	Scores   []*int // slice of pointers
	Meta     map[string]string
	MetaPtr  map[string]*string // map with pointer values
}

// Address is a nested struct.
type Address struct {
	Street string
	City   string
}

// UserDTO is the destination type.
type UserDTO struct {
	ID      int
	Name    string
	Email   string
	Age     int
	Score   *int
	Address *AddressDTO
	Tags    []string
	Scores  []*int
	Meta    map[string]string
	MetaPtr map[string]*string
}

// AddressDTO is a nested DTO.
type AddressDTO struct {
	Street string
	City   string
}

// convertgen:@converter
type UserConverter interface {
	// ConvertAddress converts Address to AddressDTO
	ConvertAddress(src *Address) *AddressDTO

	// convertgen:@map(FullName, Name)
	// convertgen:@ignore(Password)
	ConvertUser(src *User) *UserDTO

	ConvertUserSlice(src []*User) []*UserDTO
}
