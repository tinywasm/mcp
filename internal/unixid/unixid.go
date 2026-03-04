package unixid

import (
	"crypto/rand"
	"encoding/hex"
)

type UnixID struct{}

func NewUnixID() (*UnixID, error) {
	return &UnixID{}, nil
}

func (u *UnixID) GetNewID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
