package cartsess

import (
	"math/rand"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func generateID(length int) string {
	if length < 32 {
		length = 32
	}
	b := make([]rune, length)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letterRunes[rng.Intn(len(letterRunes))]
	}
	return string(b)
}
