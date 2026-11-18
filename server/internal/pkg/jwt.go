package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UID      string   `json:"uid"`
	Username string   `json:"username"`
	DeviceID string   `json:"deviceId"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

func newJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func SignToken(secret string, uid, username, deviceID string, roles []string, ttl time.Duration) (string, string, error) {
	now := time.Now()
	jti := newJTI()
	claims := Claims{
		UID: uid, Username: username, DeviceID: deviceID, Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "nexim", Subject: uid, ID: jti,
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	return s, jti, err
}

func ParseToken(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
