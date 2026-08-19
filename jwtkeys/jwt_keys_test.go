package jwtkeys_test

import (
	"os"
	"path/filepath"

	"testing"

	"github.com/user0608/bobi/jwtkeys"
)

func TestNewJwtKeyStoreGeneratesKeysWhenMissing(t *testing.T) {
	dir := t.TempDir()

	config := jwtkeys.JwtKeysConfig{
		PrivateKey: filepath.Join(dir, "jwt", "private.pem"),
		PublicKey:  filepath.Join(dir, "jwt", "public.pem"),
	}

	provider, err := jwtkeys.NewJwtKeyStore(config)
	if err != nil {
		t.Fatalf("NewJwtKeyStore() error = %v", err)
	}

	if provider.SigningKey() == nil {
		t.Fatal("SigningKey() returned nil")
	}

	if provider.VerificationKey() == nil {
		t.Fatal("VerificationKey() returned nil")
	}

	if _, err := os.Stat(config.PrivateKey); err != nil {
		t.Fatalf("private key file was not created: %v", err)
	}

	if _, err := os.Stat(config.PublicKey); err != nil {
		t.Fatalf("public key file was not created: %v", err)
	}
}

func TestNewJwtKeyStoreLoadsExistingKeys(t *testing.T) {
	dir := t.TempDir()

	config := jwtkeys.JwtKeysConfig{
		PrivateKey: filepath.Join(dir, "private.pem"),
		PublicKey:  filepath.Join(dir, "public.pem"),
	}

	firstProvider, err := jwtkeys.NewJwtKeyStore(config)
	if err != nil {
		t.Fatalf("first NewJwtKeyStore() error = %v", err)
	}

	secondProvider, err := jwtkeys.NewJwtKeyStore(config)
	if err != nil {
		t.Fatalf("second NewJwtKeyStore() error = %v", err)
	}

	if firstProvider.SigningKey().N.Cmp(secondProvider.SigningKey().N) != 0 {
		t.Fatal("expected existing private key to be loaded, got different modulus")
	}

	if firstProvider.VerificationKey().N.Cmp(secondProvider.VerificationKey().N) != 0 {
		t.Fatal("expected existing public key to be loaded, got different modulus")
	}
}

func TestNewJwtKeyStoreRegeneratesPairWhenOnlyPrivateExists(t *testing.T) {
	dir := t.TempDir()

	config := jwtkeys.JwtKeysConfig{
		PrivateKey: filepath.Join(dir, "private.pem"),
		PublicKey:  filepath.Join(dir, "public.pem"),
	}

	if err := os.MkdirAll(filepath.Dir(config.PrivateKey), 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(config.PrivateKey, []byte("invalid private key"), 0600); err != nil {
		t.Fatal(err)
	}

	provider, err := jwtkeys.NewJwtKeyStore(config)
	if err != nil {
		t.Fatalf("NewJwtKeyStore() error = %v", err)
	}

	if provider.SigningKey() == nil {
		t.Fatal("SigningKey() returned nil")
	}

	if provider.VerificationKey() == nil {
		t.Fatal("VerificationKey() returned nil")
	}
}

func TestNewJwtKeyStoreRegeneratesPairWhenOnlyPublicExists(t *testing.T) {
	dir := t.TempDir()

	config := jwtkeys.JwtKeysConfig{
		PrivateKey: filepath.Join(dir, "private.pem"),
		PublicKey:  filepath.Join(dir, "public.pem"),
	}

	if err := os.MkdirAll(filepath.Dir(config.PublicKey), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(config.PublicKey, []byte("invalid public key"), 0644); err != nil {
		t.Fatal(err)
	}

	provider, err := jwtkeys.NewJwtKeyStore(config)
	if err != nil {
		t.Fatalf("NewJwtKeyStore() error = %v", err)
	}

	if provider.SigningKey() == nil {
		t.Fatal("SigningKey() returned nil")
	}

	if provider.VerificationKey() == nil {
		t.Fatal("VerificationKey() returned nil")
	}
}

func TestNewJwtKeyStoreReturnsErrorWhenExistingPrivateKeyIsInvalid(t *testing.T) {
	dir := t.TempDir()

	config := jwtkeys.JwtKeysConfig{
		PrivateKey: filepath.Join(dir, "private.pem"),
		PublicKey:  filepath.Join(dir, "public.pem"),
	}

	if err := os.WriteFile(config.PrivateKey, []byte("invalid private key"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(config.PublicKey, []byte("invalid public key"), 0644); err != nil {
		t.Fatal(err)
	}

	provider, err := jwtkeys.NewJwtKeyStore(config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if provider != nil {
		t.Fatal("expected provider nil")
	}
}
