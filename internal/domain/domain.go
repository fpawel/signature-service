package domain

import (
	"context"
	"crypto"
	"errors"
)

type (
	SignatureDevice struct {
		ID, Algorithm, Label string
		PrivateKey           crypto.PrivateKey
		LastSignature        string
		SignatureCounter     int64
		PublicKeyPem         string
	}

	Signature struct {
		ID         string
		DeviceID   string
		Counter    int64
		Signature  string
		SignedData string
	}

	SignatureDeviceInfo struct {
		Algorithm        string
		LastSignature    string
		SignatureCounter int64
		PrivateKey       crypto.PrivateKey
	}

	Tx interface {
		CreateNewSignatureDevice(
			_ context.Context,
			id string,
			alg string,
			label string,
			privateKey crypto.PrivateKey,
			publicKeyPEM string,
		) error
		GetSignatureDeviceInfoByID(_ context.Context, deviceID string) (SignatureDeviceInfo, error)
		AddSignature(_ context.Context, deviceID string, signature string) error

		ListSignatureDevices(_ context.Context) ([]SignatureDevice, error)
		GetSignatureDevice(_ context.Context, deviceID string) (SignatureDevice, error)
		ListDeviceSignatures(_ context.Context, deviceID string) ([]Signature, error)
		GetDeviceSignature(_ context.Context, deviceID, signatureID string) (Signature, error)

		Commit(ctx context.Context) error
		Rollback(ctx context.Context) error
	}

	Repository interface {
		Tx(ctx context.Context) (Tx, error)
	}
)

var (
	ErrSignatureDeviceAlreadyExists = errors.New("signature device already exists")
	ErrSignatureDeviceNotFound      = errors.New("signature device not found")
	ErrSignatureNotFound            = errors.New("signature not found")
)
