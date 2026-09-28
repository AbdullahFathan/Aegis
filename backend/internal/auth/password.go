package auth

import "golang.org/x/crypto/bcrypt"

var bcryptCost = bcrypt.DefaultCost

func SetBcryptCost(cost int) {
	bcryptCost = cost
}

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
