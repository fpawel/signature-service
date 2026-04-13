package signing

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

type (
	Registry struct {
		Signers map[string]Signer
	}

	Signer interface {
		GenerateKeyPair() (crypto.PrivateKey, crypto.PublicKey, error)
		Sign(crypto.PrivateKey, []byte) ([]byte, error)
	}
)

var ErrUnknowAlgorithm = errors.New("unknown signing algorithm")

func NewRegistry() Registry {
	return Registry{
		Signers: make(map[string]Signer),
	}
}

func (r Registry) RegisterSigner(alg string, signer Signer) {
	r.Signers[alg] = signer
}

func (r Registry) GetSigner(alg string) (Signer, error) {
	signer, ok := r.Signers[alg]
	if !ok {
		return nil, r.newErrUnknowAlgorithm()
	}
	return signer, nil
}

func (r Registry) GenerateKeyPair(alg string) (crypto.PrivateKey, string, error) {
	signer, ok := r.Signers[alg]
	if !ok {
		return nil, "", r.newErrUnknowAlgorithm()
	}
	privateKey, publicKey, err := signer.GenerateKeyPair()
	if err != nil {
		return nil, "", fmt.Errorf("generate key pair: %w", err)
	}

	publicKeyPEM, err := EncodePublicKeyPEM(publicKey)
	if err != nil {
		return nil, "", fmt.Errorf("encode public key: %w", err)
	}
	return privateKey, publicKeyPEM, nil
}

func (r Registry) newErrUnknowAlgorithm() error {
	xs := make([]string, 0, len(r.Signers))
	for k := range r.Signers {
		xs = append(xs, k)
	}
	return fmt.Errorf("%w: must be one of %s", ErrUnknowAlgorithm, strings.Join(xs, ", "))
}

func EncodePublicKeyPEM(publicKey crypto.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}

	return string(pem.EncodeToMemory(block)), nil
}
