// Package platform tem as implementações de produção das portas que não são
// repositories: senha, relógio e IDs.
package platform

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct{}

func (Bcrypt) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func (Bcrypt) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type RandomIDs struct{}

func (RandomIDs) NewID(prefix string) string { return prefix + "_" + hex.EncodeToString(random(10)) }

func (RandomIDs) NewToken() string { return base64.RawURLEncoding.EncodeToString(random(32)) }

func random(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b) // nunca falha desde o Go 1.24
	return b
}
