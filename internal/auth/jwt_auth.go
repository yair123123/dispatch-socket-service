package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type JWTAuthenticator struct {
	secret    []byte
	publicKey *rsa.PublicKey
}

func NewJWTAuthenticator(secret, publicKeyPEM string) (*JWTAuthenticator, error) {
	a := &JWTAuthenticator{}
	if secret != "" {
		a.secret = []byte(secret)
	}
	if publicKeyPEM != "" {
		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		a.publicKey = key
	}
	return a, nil
}

func (a *JWTAuthenticator) ExtractDriverID(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("empty token")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		switch token.Method.Alg() {
		case jwt.SigningMethodHS256.Alg(), jwt.SigningMethodHS384.Alg(), jwt.SigningMethodHS512.Alg():
			if len(a.secret) == 0 {
				return nil, errors.New("jwt secret not configured")
			}
			return a.secret, nil
		case jwt.SigningMethodRS256.Alg(), jwt.SigningMethodRS384.Alg(), jwt.SigningMethodRS512.Alg():
			if a.publicKey == nil {
				return nil, errors.New("jwt public key not configured")
			}
			return a.publicKey, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	for _, candidate := range []string{"driver_id", "sub"} {
		if v, exists := claims[candidate]; exists {
			s, ok := v.(string)
			if ok && strings.TrimSpace(s) != "" {
				return s, nil
			}
		}
	}
	return "", errors.New("driver_id not found")
}
