package signing

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
)

type RSASigner struct{}

func (RSASigner) GenerateKeyPair() (crypto.PrivateKey, crypto.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func (RSASigner) Sign(privateKey crypto.PrivateKey, data []byte) ([]byte, error) {
	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unexpected RSA private key type %T", privateKey)
	}

	hash := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])
}
