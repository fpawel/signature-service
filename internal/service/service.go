package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/fpawel/signature-service/internal/api/swagger/generated/models"
	"github.com/fpawel/signature-service/internal/api/swagger/generated/restapi/operations"
	"github.com/fpawel/signature-service/internal/domain"
	"github.com/fpawel/signature-service/internal/signing"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

type (
	Service struct {
		domain.Repository
		signing.Registry
		Now func() time.Time
	}
)

func NewService(rep domain.Repository, sining signing.Registry) *Service {
	return &Service{
		Repository: rep,
		Registry:   sining,
		Now:        time.Now().UTC,
	}
}

func (s *Service) CreateSignatureDevice(p operations.CreateSignatureDeviceParams) middleware.Responder {
	id := p.Body.ID
	label := p.Body.Label
	alg := p.Body.Algorithm

	privateKey, publicKeyPEM, err := s.Registry.GenerateKeyPair(alg)
	if err != nil {
		if errors.Is(err, signing.ErrUnknowAlgorithm) {
			return operations.NewCreateSignatureDeviceBadRequest().WithPayload(err.Error())
		}
		return newCreateSignatureDeviceInternalServerError(err)
	}

	ctx := p.HTTPRequest.Context()
	tx, err := s.Repository.Tx(ctx)
	if err != nil {
		return newCreateSignatureDeviceInternalServerError(err)
	}
	if err = tx.CreateNewSignatureDevice(ctx, string(id), alg, label, privateKey, publicKeyPEM); err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, domain.ErrSignatureDeviceAlreadyExists) {
			return operations.NewCreateSignatureDeviceConflict().WithPayload(err.Error())
		}
		return newCreateSignatureDeviceInternalServerError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		return newCreateSignatureDeviceInternalServerError(err)
	}

	return operations.NewCreateSignatureDeviceCreated().WithPayload(&models.SignatureDeviceResponse{
		ID:           id,
		PublicKeyPem: publicKeyPEM,
	})
}

func (s *Service) SignTransaction(p operations.SignTransactionParams) middleware.Responder {
	ctx := p.HTTPRequest.Context()
	tx, err := s.Repository.Tx(ctx)
	if err != nil {
		return newSignTransactionInternalServerError(err)
	}

	dev, err := tx.GetSignatureDeviceInfoByID(ctx, string(p.DeviceID))
	if err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, domain.ErrSignatureDeviceNotFound) {
			return operations.NewSignTransactionNotFound().WithPayload(err.Error())
		}
		return newSignTransactionInternalServerError(err)
	}

	signerImpl, err := s.GetSigner(dev.Algorithm)
	if err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, signing.ErrUnknowAlgorithm) {
			return operations.NewSignTransactionBadRequest().WithPayload(err.Error())
		}
		return newSignTransactionInternalServerError(err)

	}

	securedData := fmt.Sprintf("%d_%s_%s", dev.SignatureCounter, p.Body.Data, dev.LastSignature)
	signatureBytes, err := signerImpl.Sign(dev.PrivateKey, []byte(securedData))
	if err != nil {
		_ = tx.Rollback(ctx)
		return newSignTransactionInternalServerError(err)
	}

	if err = tx.AddSignature(ctx, string(p.DeviceID), base64.StdEncoding.EncodeToString(signatureBytes)); err != nil {
		_ = tx.Rollback(ctx)
		return newSignTransactionInternalServerError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		return newSignTransactionInternalServerError(err)
	}

	return operations.NewSignTransactionCreated().WithPayload(&models.SignatureResponse{
		Signature:  base64.StdEncoding.EncodeToString(signatureBytes),
		SignedData: securedData,
	})
}

func (s *Service) ListSignatureDevices(p operations.ListSignatureDevicesParams) middleware.Responder {
	ctx := p.HTTPRequest.Context()

	tx, err := s.Repository.Tx(ctx)
	if err != nil {
		return operations.NewListSignatureDevicesInternalServerError().WithPayload(err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()

	devices, err := tx.ListSignatureDevices(ctx)
	if err != nil {
		return operations.NewListSignatureDevicesInternalServerError().WithPayload(err.Error())
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	items := make([]*models.SignatureDevice, 0, len(devices))
	for _, dev := range devices {
		item := newSignatureDeviceModel(dev, dev.PublicKeyPem)
		items = append(items, item)
	}

	return operations.NewListSignatureDevicesOK().WithPayload(&models.SignatureDeviceListResponse{
		Items: items,
	})
}

func (s *Service) GetSignatureDevice(p operations.GetSignatureDeviceParams) middleware.Responder {
	ctx := p.HTTPRequest.Context()

	tx, err := s.Repository.Tx(ctx)
	if err != nil {
		return operations.NewGetSignatureDeviceInternalServerError().WithPayload(err.Error())
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	dev, err := tx.GetSignatureDevice(ctx, string(p.DeviceID))
	if err != nil {
		if errors.Is(err, domain.ErrSignatureDeviceNotFound) {
			return operations.NewGetSignatureDeviceNotFound().WithPayload(err.Error())
		}
		return operations.NewGetSignatureDeviceInternalServerError().WithPayload(err.Error())
	}

	item := newSignatureDeviceModel(dev, dev.PublicKeyPem)

	return operations.NewGetSignatureDeviceOK().WithPayload(item)
}

func (s *Service) ListDeviceSignatures(p operations.ListDeviceSignaturesParams) middleware.Responder {
	ctx := p.HTTPRequest.Context()

	tx, err := s.Repository.Tx(ctx)
	if err != nil {
		return operations.NewListDeviceSignaturesInternalServerError().WithPayload(err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sigs, err := tx.ListDeviceSignatures(ctx, string(p.DeviceID))
	if err != nil {
		if errors.Is(err, domain.ErrSignatureDeviceNotFound) {
			return operations.NewListDeviceSignaturesNotFound().WithPayload(err.Error())
		}
		return operations.NewListDeviceSignaturesInternalServerError().WithPayload(err.Error())
	}

	items := make([]*models.Signature, 0, len(sigs))
	for _, sig := range sigs {
		items = append(items, newSignatureModel(sig))
	}

	return operations.NewListDeviceSignaturesOK().WithPayload(&models.SignatureListResponse{
		Items: items,
	})
}

func (s *Service) GetDeviceSignature(p operations.GetDeviceSignatureParams) middleware.Responder {
	ctx := p.HTTPRequest.Context()

	tx, err := s.Repository.Tx(ctx)
	if err != nil {
		return operations.NewGetDeviceSignatureInternalServerError().WithPayload(err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sig, err := tx.GetDeviceSignature(ctx, string(p.DeviceID), string(p.SignatureID))
	if err != nil {
		if errors.Is(err, domain.ErrSignatureDeviceNotFound) || errors.Is(err, domain.ErrSignatureNotFound) {
			return operations.NewGetDeviceSignatureNotFound().WithPayload(err.Error())
		}
		return operations.NewGetDeviceSignatureInternalServerError().WithPayload(err.Error())
	}

	return operations.NewGetDeviceSignatureOK().WithPayload(newSignatureModel(sig))
}

func newSignatureDeviceModel(dev domain.SignatureDevice, publicKeyPEM string) *models.SignatureDevice {
	return &models.SignatureDevice{
		ID:               strfmt.UUID(dev.ID),
		Algorithm:        dev.Algorithm,
		Label:            dev.Label,
		SignatureCounter: dev.SignatureCounter,
		LastSignature:    dev.LastSignature,
		PublicKeyPem:     publicKeyPEM,
	}
}

func newSignatureModel(sig domain.Signature) *models.Signature {
	return &models.Signature{
		ID:         strfmt.UUID(sig.ID),
		DeviceID:   strfmt.UUID(sig.DeviceID),
		Counter:    sig.Counter,
		Signature:  sig.Signature,
		SignedData: sig.SignedData,
	}
}

func newSignTransactionInternalServerError(err error) middleware.Responder {
	return operations.NewSignTransactionInternalServerError().WithPayload(err.Error())
}

func newCreateSignatureDeviceInternalServerError(err error) middleware.Responder {
	return operations.NewCreateSignatureDeviceInternalServerError().WithPayload(err.Error())
}
