package jwtkeys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// JwtKeyStore holds the RSA keys used to sign and verify JWTs.
type JwtKeyStore struct {
	signingKey      *rsa.PrivateKey
	verificationKey *rsa.PublicKey
}

// NewJwtKeyStore loads an existing RSA key pair or generates one when either key is missing.
func NewJwtKeyStore(config JwtKeysConfig) (*JwtKeyStore, error) {
	config = config.withDefaults()
	signingKeyPath := config.PrivateKey
	verificationKeyPath := config.PublicKey

	if err := ensureRSAKeyPairFiles(signingKeyPath, verificationKeyPath); err != nil {
		return nil, err
	}

	signingKeyPEM, err := os.ReadFile(signingKeyPath)
	if err != nil {
		return nil, err
	}

	verificationKeyPEM, err := os.ReadFile(verificationKeyPath)
	if err != nil {
		return nil, err
	}

	signingKey, err := gojwt.ParseRSAPrivateKeyFromPEM(signingKeyPEM)
	if err != nil {
		return nil, err
	}

	verificationKey, err := gojwt.ParseRSAPublicKeyFromPEM(verificationKeyPEM)
	if err != nil {
		return nil, err
	}

	return &JwtKeyStore{
		signingKey:      signingKey,
		verificationKey: verificationKey,
	}, nil
}

func (s *JwtKeyStore) SigningKey() *rsa.PrivateKey {
	return s.signingKey
}

func (s *JwtKeyStore) VerificationKey() *rsa.PublicKey {
	return s.verificationKey
}

func ensureRSAKeyPairFiles(signingKeyPath, verificationKeyPath string) error {
	if err := os.MkdirAll(filepath.Dir(signingKeyPath), 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(verificationKeyPath), 0755); err != nil {
		return err
	}

	_, signingKeyErr := os.Stat(signingKeyPath)
	_, verificationKeyErr := os.Stat(verificationKeyPath)

	if signingKeyErr == nil && verificationKeyErr == nil {
		return nil
	}

	if signingKeyErr != nil && !errors.Is(signingKeyErr, os.ErrNotExist) {
		return signingKeyErr
	}

	if verificationKeyErr != nil && !errors.Is(verificationKeyErr, os.ErrNotExist) {
		return verificationKeyErr
	}

	if err := os.Remove(signingKeyPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Remove(verificationKeyPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	signingKeyFile, err := os.OpenFile(signingKeyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer signingKeyFile.Close()

	verificationKeyFile, err := os.OpenFile(verificationKeyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer verificationKeyFile.Close()

	keyPair, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	if err := keyPair.Validate(); err != nil {
		return fmt.Errorf("invalid RSA key pair: %w", err)
	}

	signingKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(keyPair),
	}

	if err := pem.Encode(signingKeyFile, signingKeyBlock); err != nil {
		return fmt.Errorf("encode signing key: %w", err)
	}

	verificationKeyDER, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal verification key: %w", err)
	}

	verificationKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: verificationKeyDER,
	}

	if err := pem.Encode(verificationKeyFile, verificationKeyBlock); err != nil {
		return fmt.Errorf("encode verification key: %w", err)
	}

	return nil
}
