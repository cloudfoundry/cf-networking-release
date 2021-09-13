package warrant

// Member is the representation of a group member resource within UAA.
// This is probably just a user.
type Member struct {
	Origin string
	Type   string
	Value  string
}
