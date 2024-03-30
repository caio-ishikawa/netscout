package shared

// Checks if string exists in a slice
func SliceContains(slice []string, toCheck string) bool {
	for _, x := range slice {
		if x == toCheck {
			return true
		}
	}

	return false
}
