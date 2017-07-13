// Copyright 2016 ISRG.  All rights reserved
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package wrappers wraps the GRPC calls in the core interfaces.
package grpc

import (
	"crypto/x509"
	"errors"
	"time"

	"golang.org/x/net/context"

	caPB "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/revocation"
)

// CertificateAuthorityClientWrapper is the gRPC version of a
// core.CertificateAuthority client. It composites a CertificateAuthorityClient
// and OCSPGeneratorClient, either of which may be nil if the calling code
// doesn't intend to use the relevant functions. Once we've fully moved to gRPC,
// calling code will do away with this wrapper and directly instantiate exactly
// the type of client it needs.
type CertificateAuthorityClientWrapper struct {
	inner     caPB.CertificateAuthorityClient
	innerOCSP caPB.OCSPGeneratorClient
}

func NewCertificateAuthorityClient(inner caPB.CertificateAuthorityClient, innerOCSP caPB.OCSPGeneratorClient) *CertificateAuthorityClientWrapper {
	return &CertificateAuthorityClientWrapper{inner, innerOCSP}
}

func (cac CertificateAuthorityClientWrapper) IssueCertificate(ctx context.Context, csr x509.CertificateRequest, regID int64) (core.Certificate, error) {
	if cac.inner == nil {
		return core.Certificate{}, errors.New("this CA client does not support issuing certificates")
	}
	res, err := cac.inner.IssueCertificate(ctx, &caPB.IssueCertificateRequest{
		Csr:            csr.Raw,
		RegistrationID: &regID,
	})
	if err != nil {
		return core.Certificate{}, err
	}
	return pbToCert(res), nil
}

func (cac CertificateAuthorityClientWrapper) GenerateOCSP(ctx context.Context, ocspReq core.OCSPSigningRequest) ([]byte, error) {
	if cac.innerOCSP == nil {
		return nil, errors.New("this CA client does not support generating OCSP")
	}
	reason := int32(ocspReq.Reason)
	revokedAt := ocspReq.RevokedAt.UnixNano()
	res, err := cac.innerOCSP.GenerateOCSP(ctx, &caPB.GenerateOCSPRequest{
		CertDER:   ocspReq.CertDER,
		Status:    &ocspReq.Status,
		Reason:    &reason,
		RevokedAt: &revokedAt,
	})
	if err != nil {
		return nil, err
	}
	return res.Response, nil
}
func (cac CertificateAuthorityClientWrapper) ChangeValidity(newVal string) {

}

// CertificateAuthorityServerWrapper is the gRPC version of a core.CertificateAuthority server
type CertificateAuthorityServerWrapper struct {
	inner core.CertificateAuthority
}

func NewCertificateAuthorityServer(inner core.CertificateAuthority) *CertificateAuthorityServerWrapper {
	return &CertificateAuthorityServerWrapper{inner}
}

func (cas *CertificateAuthorityServerWrapper) IssueCertificate(ctx context.Context, request *caPB.IssueCertificateRequest) (*corepb.Certificate, error) {
	csr, err := x509.ParseCertificateRequest(request.Csr)
	if err != nil {
		return nil, err
	}
	cert, err := cas.inner.IssueCertificate(ctx, *csr, *request.RegistrationID)
	if err != nil {
		return nil, err
	}
	return certToPB(cert), nil
}

func (cas *CertificateAuthorityServerWrapper) GenerateOCSP(ctx context.Context, request *caPB.GenerateOCSPRequest) (*caPB.OCSPResponse, error) {
	res, err := cas.inner.GenerateOCSP(ctx, core.OCSPSigningRequest{
		CertDER:   request.CertDER,
		Status:    *request.Status,
		Reason:    revocation.Reason(*request.Reason),
		RevokedAt: time.Unix(0, *request.RevokedAt),
	})
	if err != nil {
		return nil, err
	}
	return &caPB.OCSPResponse{Response: res}, nil
}
