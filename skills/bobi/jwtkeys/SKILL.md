---
name: jwtkeys
description: Use Bobi jwtkeys for RSA JWT signing and verification key loading or generation. Trigger when configuring JwtKeysConfig, NewJwtKeyStore, SigningKey, or VerificationKey.
---

# Bobi JWT Keys

`jwtkeys.NewJwtKeyStore` loads an RSA key pair from PEM files and generates a 2048-bit pair when either file is missing:

```go
store, err := jwtkeys.NewJwtKeyStore(jwtkeys.JwtKeysConfig{
	PrivateKey: "./keys/private.pem",
	PublicKey:  "./keys/public.pem",
})
if err != nil {
	return err
}

token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
signed, err := token.SignedString(store.SigningKey())
```

Defaults are `./keys/private.pem` and `./keys/public.pem`. The private key is created with restrictive permissions. Treat it as a secret: never commit it, log it, or expose it through an HTTP response.

Use `VerificationKey()` for parsing/verifying tokens and explicitly restrict accepted signing algorithms in the JWT parser. Bobi manages key material only; it does not create claims, tokens, middleware, refresh flows, or authorization policy.
