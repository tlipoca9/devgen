package testdata

// markgen:@mark
// goplugin:@mark
type User struct {
	ID   int
	Name string
}

// markgen:@mark
// goplugin:@mark
type Order struct {
	ID     int
	UserID int
	Amount float64
}

// This type is NOT marked
type Config struct {
	Debug bool
}
