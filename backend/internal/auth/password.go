package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword hashes a plaintext password using bcrypt with the given cost.
// Cost is configurable (minimum 12 per spec) and passed in from config — not
// hardcoded here — so it can be lowered in tests for speed without changing production behavior.
func HashPassword(plain string, cost int) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePassword checks whether a plaintext password matches the stored bcrypt hash.
// Returns nil if they match, bcrypt.ErrMismatchedHashAndPassword otherwise.
func ComparePassword(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
