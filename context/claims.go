package context

// "Set" data type for convenient claims handling
type claimSet map[string]struct{}

// Check if claim exist
func (claims claimSet) Has(claim string) bool {
	_, exists := claims[claim]
	return exists
}
