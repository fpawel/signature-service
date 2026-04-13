package inmem

import (
	"context"
	"crypto"
	"encoding/base64"
	"sort"
	"sync"

	"github.com/fpawel/signature-service/internal/domain"
)

type (
	Storage struct {
		mu         sync.RWMutex
		devices    map[string]domain.SignatureDevice
		signatures map[string][]domain.Signature
	}
	Tx struct {
		Storage *Storage
	}
)

func NewStorage() *Storage {
	return &Storage{
		devices: make(map[string]domain.SignatureDevice),
	}
}

func (s *Storage) Tx(context.Context) (domain.Tx, error) {
	s.mu.Lock()
	return &Tx{Storage: s}, nil
}

func (tx *Tx) Commit(context.Context) error {
	tx.Storage.mu.Unlock()
	return nil
}
func (tx *Tx) Rollback(context.Context) error {
	tx.Storage.mu.Unlock()
	return nil
}

func (tx *Tx) GetSignatureDeviceByID(_ context.Context, id string) (domain.SignatureDevice, error) {
	d, ok := tx.Storage.devices[id]
	if ok {
		return domain.SignatureDevice{}, domain.ErrSignatureDeviceAlreadyExists
	}
	return d, nil
}

func (tx *Tx) GetSignatureDeviceInfoByID(_ context.Context, deviceID string) (domain.SignatureDeviceInfo, error) {
	device, ok := tx.Storage.devices[deviceID]
	if !ok {
		return domain.SignatureDeviceInfo{}, domain.ErrSignatureDeviceNotFound
	}
	return domain.SignatureDeviceInfo{
		Algorithm:        device.Algorithm,
		LastSignature:    device.LastSignature,
		SignatureCounter: device.SignatureCounter,
		PrivateKey:       device.PrivateKey,
	}, nil

}

func (tx *Tx) AddSignature(_ context.Context, deviceID string, signature string) error {
	dev := tx.Storage.devices[deviceID]
	dev.SignatureCounter++
	dev.LastSignature = signature
	tx.Storage.devices[deviceID] = dev
	return nil
}

func (tx *Tx) CreateNewSignatureDevice(
	_ context.Context,
	id string,
	alg string,
	label string,
	privateKey crypto.PrivateKey,
	publicKeyPEM string,
) error {
	if _, ok := tx.Storage.devices[id]; ok {
		return domain.ErrSignatureDeviceAlreadyExists
	}

	tx.Storage.devices[id] = domain.SignatureDevice{
		ID:            id,
		Algorithm:     alg,
		Label:         label,
		PrivateKey:    privateKey,
		LastSignature: base64.StdEncoding.EncodeToString([]byte(id)),
		PublicKeyPem:  publicKeyPEM,
	}
	return nil
}

func (tx *Tx) ListSignatureDevices(_ context.Context) ([]domain.SignatureDevice, error) {
	devices := make([]domain.SignatureDevice, 0, len(tx.Storage.devices))
	for _, dev := range tx.Storage.devices {
		devices = append(devices, dev)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	return devices, nil
}

func (tx *Tx) GetSignatureDevice(_ context.Context, deviceID string) (domain.SignatureDevice, error) {
	dev, ok := tx.Storage.devices[deviceID]
	if !ok {
		return domain.SignatureDevice{}, domain.ErrSignatureDeviceNotFound
	}
	return dev, nil
}

func (tx *Tx) ListDeviceSignatures(_ context.Context, deviceID string) ([]domain.Signature, error) {
	if _, ok := tx.Storage.devices[deviceID]; !ok {
		return nil, domain.ErrSignatureDeviceNotFound
	}

	src := tx.Storage.signatures[deviceID]
	dst := make([]domain.Signature, len(src))
	copy(dst, src)

	return dst, nil
}

func (tx *Tx) GetDeviceSignature(_ context.Context, deviceID, signatureID string) (domain.Signature, error) {
	if _, ok := tx.Storage.devices[deviceID]; !ok {
		return domain.Signature{}, domain.ErrSignatureDeviceNotFound
	}

	for _, sig := range tx.Storage.signatures[deviceID] {
		if sig.ID == signatureID {
			return sig, nil
		}
	}

	return domain.Signature{}, domain.ErrSignatureNotFound
}
