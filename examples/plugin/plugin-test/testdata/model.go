package testdata

// markgen:@mark
// execgen:@mark
// goplugin:@mark
type User struct {
	ID   int
	Name string
}

// markgen:@mark
// execgen:@mark
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
