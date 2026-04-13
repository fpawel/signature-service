package signing

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"fmt"
	"math/big"
)

type ECDSASigner struct{}

func (ECDSASigner) GenerateKeyPair() (crypto.PrivateKey, crypto.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func (ECDSASigner) Sign(privateKey crypto.PrivateKey, data []byte) ([]byte, error) {
	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unexpected ECDSA private key type %T", privateKey)
	}

	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, ecdsaKey, hash[:])
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(struct {
		R *big.Int
		S *big.Int
	}{R: r, S: s})
}
