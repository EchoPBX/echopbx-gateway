package jwt

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type Validator struct {
	keys     []*x509.Certificate
	iss, aud string
}

func NewValidator(pubPemPaths []string, issuer, audience string) (*Validator, error) {
	var certs []*x509.Certificate
	for _, p := range pubPemPaths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(b)
		if block == nil {
			return nil, errors.New("invalid pem")
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return &Validator{keys: certs, iss: issuer, aud: audience}, nil
}

func (v *Validator) Verify(tokenStr string) (jwt.MapClaims, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		for _, c := range v.keys {
			if c.Subject.CommonName == kid {
				return c.PublicKey, nil
			}
		}
		return v.keys[0].PublicKey, nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	claims, _ := tok.Claims.(jwt.MapClaims)
	if v.iss != "" && claims["iss"] != v.iss {
		return nil, errors.New("iss mismatch")
	}
	if v.aud != "" && claims["aud"] != v.aud {
		return nil, errors.New("aud mismatch")
	}
	return claims, nil
}
