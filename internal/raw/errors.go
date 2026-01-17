package raw

import "fmt"

// KeyNotFoundError represents an error when a key is not found in a collection
type KeyNotFoundError struct {
	Key        string
	Collection string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("key not found: %s in %s", e.Key, e.Collection)
}

// NewKeyNotFoundError creates a new KeyNotFoundError
func NewKeyNotFoundError(key, collection string) error {
	return KeyNotFoundError{Key: key, Collection: collection}
}
