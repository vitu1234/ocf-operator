// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package ocfclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	enjson "encoding/json"
	"fmt"
	"os"
	"time"

	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"

	local "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/device/v2/pkg/security/signer"
	"github.com/plgd-dev/kit/v2/security"
	Options "github.com/vitu1234/ocf-operator/api/v1alpha1"

	// "github.com/jessevdk/go-flags"

	"github.com/plgd-dev/device/v2/schema/interfaces"
)

const Timeout = time.Second * 10

type (
	// OCFClient is an OCF Client for working with devices
	OCFClient struct {
		client  *local.Client
		devices []local.DeviceDetails
	}
)

var (
	CertIdentity = "00000000-0000-0000-0000-000000000001"

	MfgCert         = []byte{}
	MfgKey          = []byte{}
	MfgTrustedCA    = []byte{}
	MfgTrustedCAKey = []byte{}

	IdentityTrustedCA         = []byte{}
	IdentityTrustedCAKey      = []byte{}
	IdentityIntermediateCA    = []byte{}
	IdentityIntermediateCAKey = []byte{}
	IdentityCert              = []byte{}
	IdentityKey               = []byte{}
)

const (
	validFromNow = "now"
	validForYear = 8760 * time.Hour
)

type SetupSecureClient struct {
	ca      []*x509.Certificate
	mfgCA   []*x509.Certificate
	mfgCert tls.Certificate
}

func (c *SetupSecureClient) GetManufacturerCertificate() (tls.Certificate, error) {
	if c.mfgCert.PrivateKey == nil {
		return c.mfgCert, fmt.Errorf("private key not set")
	}
	return c.mfgCert, nil
}

func (c *SetupSecureClient) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.mfgCA) == 0 {
		return nil, fmt.Errorf("certificate authority not set")
	}
	return c.mfgCA, nil
}

func (c *SetupSecureClient) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.ca) == 0 {
		return nil, fmt.Errorf("certificate authorities not set")
	}
	return c.ca, nil
}

// Initialize creates and initializes new local client
func (c *OCFClient) Initialize() error {
	localClient, err := NewSecureClient()
	if err != nil {
		return err
	}
	c.client = localClient
	return nil
}

func (c *OCFClient) Close() error {
	if c.client != nil {
		return c.client.Close(context.Background())
	}
	return nil
}

// Discover devices in the local area
func (c *OCFClient) Discover(discoveryTimeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()
	res, err := c.client.GetDevicesDetails(ctx)
	if err != nil {
		return "", err
	}

	deviceInfo := []interface{}{}
	devices := []local.DeviceDetails{}
	for _, device := range res {
		if device.IsSecured && device.Ownership != nil {
			devices = append(devices, device)
			devInfo := map[string]interface{}{
				"id": device.ID, "name": device.Ownership.Name, "owned": device.Ownership.Owned,
				"ownerID": device.Ownership.OwnerID, "details": device.Details,
			}
			deviceInfo = append(deviceInfo, devInfo)
		}
	}
	c.devices = devices

	devicesJSON, err := enjson.MarshalIndent(deviceInfo, "", "    ")
	if err != nil {
		return "", err
	}
	return string(devicesJSON), nil
}

// OwnDevice transfers the ownership of the device to user represented by the token
func (c *OCFClient) OwnDevice(deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.client.OwnDevice(ctx, deviceID, local.WithOTMs([]local.OTMType{local.OTMType_JustWorks}))
}

// GetResources returns all resources info of the device
func (c *OCFClient) GetResources(deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	_, links, err := c.client.GetDeviceByMulticast(ctx, deviceID)
	if err != nil {
		return "", err
	}
	resourcesInfo := []map[string]interface{}{}
	for _, link := range links {
		info := map[string]interface{}{"href": link.Href} // , "rt":link.ResourceTypes, "if":link.Interfaces}
		resourcesInfo = append(resourcesInfo, info)
	}

	linksJSON, err := enjson.MarshalIndent(resourcesInfo, "", "    ")
	if err != nil {
		return "", err
	}
	return string(linksJSON), nil
}

// GetResource returns info of the resource at the given href of the device
func (c *OCFClient) GetResource(deviceID, href string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	var got interface{} // map[string]interface{}
	opts := []local.GetOption{local.WithInterface(interfaces.OC_IF_BASELINE)}
	err := c.client.GetResource(ctx, deviceID, href, &got, opts...)
	if err != nil {
		return "", err
	}

	var resourceJSON bytes.Buffer
	resourceBytes, err := json.Encode(got)
	if err != nil {
		return "", err
	}
	err = enjson.Indent(&resourceJSON, resourceBytes, "", "    ")
	if err != nil {
		return "", err
	}
	return resourceJSON.String(), nil
}

// UpdateResource updates a resource of the device
func (c *OCFClient) UpdateResource(deviceID string, href string, data interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	// opts := []local.UpdateOption{local.WithInterface(interfaces.OC_IF_BASELINE)}
	return c.client.UpdateResource(ctx, deviceID, href, data, nil)
}

// DisownDevice removes the current ownership
func (c *OCFClient) DisownDevice(deviceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.client.DisownDevice(ctx, deviceID)
}

func loadCertificateIdentity(certIdentity string) {
	if certIdentity == "" {
		return
	}
	CertIdentity = certIdentity
}

func loadMfgTrustCA(mTrustCA string) {
	if mTrustCA == "" {
		return
	}
	mfgTrustCA, err := os.ReadFile(mTrustCA)
	if err != nil {
		fmt.Println("Unable to read Manufacturer Trust CA's Certificate: " + err.Error())
		return
	}
	fmt.Println("Reading Manufacturer Trust CA's Certificate from " + mTrustCA + " was successful.")
	MfgTrustedCA = mfgTrustCA
}

func loadMfgTrustCAKey(mTrustCAKey string) {
	if mTrustCAKey == "" {
		return
	}
	mfgTrustCAKey, err := os.ReadFile(mTrustCAKey)
	if err != nil {
		fmt.Println("Unable to read Manufacturer Trust CA's Private Key: " + err.Error())
		return
	}
	fmt.Println("Reading Manufacturer Trust CA's Private Key from " + mTrustCAKey + " was successful.")
	MfgTrustedCAKey = mfgTrustCAKey
}

func generateMfgTrustCA(mTrustCA, mTrustCAKey string) {
	if mTrustCA == "" || mTrustCAKey == "" {
		outCert := "mfg_rootca.crt"
		outKey := "mfg_rootca.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.Subject.CommonName = "TEST Mfg ROOT CA"
			cfg.BasicConstraints.MaxPathLen = -1
			cfg.ValidFrom = validFromNow
			cfg.ValidFor = validForYear

			err := generateRootCA(cfg, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Manufacturer Trust CA: " + err.Error())
			} else {
				fmt.Println("Generating Manufacturer Trust CA to " + outCert + ", " + outKey + " was successful.")
			}
		}
	}
}

func ReadCommandOptions(opts Options.Options) {
	// fmt.Println("Usage of OCF Client Options :")
	// fmt.Println("    --discoveryTimeout=<Duration>                              i.e. 5s")
	// fmt.Println("    --certIdentity=<Device UUID>                               i.e. 00000000-0000-0000-0000-000000000001")
	// fmt.Println("    --mfgCert=<Manufacturer Certificate>                       i.e. mfg_cert.crt")
	// fmt.Println("    --mfgKey=<Manufacturer Private Key>                        i.e. mfg_cert.key")
	// fmt.Println("    --mfgTrustCA=<Manufacturer Trusted CA Certificate>         i.e. mfg_rootca.crt")
	// fmt.Println("    --mfgTrustCAKey=<Manufacturer Trusted CA Private Key>      i.e. mfg_rootca.key")
	// fmt.Println("    --identityCert=<Identity Certificate>                      i.e. end_cert.crt")
	// fmt.Println("    --identityKey=<Identity Certificate>                       i.e. end_cert.key")
	// fmt.Println("    --identityIntermediateCA=<Identity Intermediate CA Certificate>     i.e. subca_cert.crt")
	// fmt.Println("    --identityIntermediateCAKey=<Identity Intermediate CA Private Key>  i.e. subca_cert.key")
	// fmt.Println("    --identityTrustCA=<Identity Trusted CA Certificate>        i.e. rootca_cert.crt")
	// fmt.Println("    --identityTrustCA=<Identity Trusted CA Private Key>        i.e. rootca_cert.key")
	// fmt.Println()

	// Load certificate identity
	loadCertificateIdentity(opts.CertIdentity)

	// Load mfg root CA certificate and private key
	loadMfgTrustCA(opts.MfgTrustCA)
	loadMfgTrustCAKey(opts.MfgTrustCAKey)

	// Generate mfg root CA certificate and private key if they don't exist
	generateMfgTrustCA(opts.MfgTrustCA, opts.MfgTrustCAKey)

	// Load mfg certificate and private key
	if opts.MfgCert != "" && opts.MfgKey != "" {
		mfgCert, err := os.ReadFile(opts.MfgCert)
		if err != nil {
			fmt.Println("Unable to read Manufacturer Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Manufacturer Certificate from " + opts.MfgCert + " was successful.")
			MfgCert = mfgCert
		}
		mfgKey, err := os.ReadFile(opts.MfgKey)
		if err != nil {
			fmt.Println("Unable to read Manufacturer Certificate's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Manufacturer Certificate's Private Key from " + opts.MfgKey + " was successful.")
			MfgKey = mfgKey
		}
	}

	// Generate mfg certificate and private key if not exists
	if opts.MfgCert == "" || opts.MfgKey == "" {
		outCert := "mfg_cert.crt"
		outKey := "mfg_cert.key"
		signerCert := "mfg_rootca.crt"
		signerKey := "mfg_rootca.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.ValidFrom = validFromNow
			cfg.ValidFor = validForYear

			err := generateIdentityCertificate(cfg, CertIdentity, signerCert, signerKey, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Manufacturer Certificate: " + err.Error())
			} else {
				fmt.Println("Generating Manufacturer Certificate to " + outCert + ", " + outKey + " was successful.")
			}
		}
	}

	// Load identity trust CA certificate and private key
	if opts.IdentityTrustCA != "" {
		identityTrustCA, err := os.ReadFile(opts.IdentityTrustCA)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Certificate from " + opts.IdentityTrustCA + " was successful.")
			IdentityTrustedCA = identityTrustCA
		}
	}

	if opts.IdentityTrustCAKey != "" {
		identityTrustCAKey, err := os.ReadFile(opts.IdentityTrustCAKey)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Private Key from " + opts.IdentityTrustCAKey + " was successful.")
			IdentityTrustedCAKey = identityTrustCAKey
		}
	}

	// Generate identity trust CA certificate and private key if not exists
	if opts.IdentityTrustCA == "" || opts.IdentityTrustCAKey == "" {
		outCert := "rootca_cert.crt"
		outKey := "rootca_cert.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.Subject.CommonName = "TEST ROOT CA"
			cfg.BasicConstraints.MaxPathLen = -1
			cfg.ValidFrom = validFromNow
			cfg.ValidFor = validForYear

			err := generateRootCA(cfg, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Identity Trust CA: " + err.Error())
			} else {
				fmt.Println("Generating Identity Trust CA to " + outCert + ", " + outKey + " was successful.")
			}
		}

		identityTrustCA, err := os.ReadFile(opts.IdentityTrustCA)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Certificate from " + opts.IdentityTrustCA + " was successful.")
			IdentityTrustedCA = identityTrustCA
		}
		identityTrustCAKey, err := os.ReadFile(opts.IdentityTrustCAKey)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Private Key from " + opts.IdentityTrustCAKey + " was successful.")
			IdentityTrustedCAKey = identityTrustCAKey
		}
	}

	// Load identity intermediate CA certificate and private key
	if opts.IdentityIntermediateCA != "" {
		identityIntermediateCA, err := os.ReadFile(opts.IdentityIntermediateCA)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Certificate from " + opts.IdentityIntermediateCA + " was successful.")
			IdentityIntermediateCA = identityIntermediateCA
		}
	}

	if opts.IdentityIntermediateCAKey != "" {
		identityIntermediateCAKey, err := os.ReadFile(opts.IdentityIntermediateCAKey)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Private Key from " + opts.IdentityIntermediateCAKey + " was successful.")
			IdentityIntermediateCAKey = identityIntermediateCAKey
		}
	}

	// Generate identity intermediate CA certificate and private key if not exists
	if opts.IdentityIntermediateCA == "" || opts.IdentityIntermediateCAKey == "" {
		outCert := "subca_cert.crt"
		outKey := "subca_cert.key"
		signerCert := "rootca_cert.crt"
		signerKey := "rootca_cert.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.Subject.CommonName = "TEST Intermediate CA"
			cfg.BasicConstraints.MaxPathLen = -1
			cfg.ValidFrom = validFromNow
			cfg.ValidFor = validForYear

			err := generateIntermediateCertificate(cfg, signerCert, signerKey, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Identity Intermediate CA: " + err.Error())
			} else {
				fmt.Println("Generating Identity Intermediate CA to " + outCert + ", " + outKey + " was successful.")
			}
		}

		identityIntermediateCA, err := os.ReadFile(outCert)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Certificate from " + outCert + " was successful.")
			IdentityIntermediateCA = identityIntermediateCA
		}
		identityIntermediateCAKey, err := os.ReadFile(outKey)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Private Key from " + outKey + " was successful.")
			IdentityIntermediateCAKey = identityIntermediateCAKey
		}
	}

	// Load identity certificate and private key
	if opts.IdentityCert != "" && opts.IdentityKey != "" {
		identityCert, err := os.ReadFile(opts.IdentityCert)
		if err != nil {
			fmt.Println("Unable to read Identity Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Certificate from " + opts.IdentityCert + " was successful.")
			IdentityCert = identityCert
		}
		identityKey, err := os.ReadFile(opts.IdentityKey)
		if err != nil {
			fmt.Println("Unable to read Identity Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Private Key from " + opts.IdentityKey + " was successful.")
			IdentityKey = identityKey
		}
	}

	// Generate identity certificate and private key if not exists
	if opts.IdentityCert == "" || opts.IdentityKey == "" {
		outCert := "end_cert.crt"
		outKey := "end_cert.key"
		signerCert := "subca_cert.crt"
		signerKey := "subca_cert.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			certConfig := generateCertificate.Configuration{}
			err := generateIdentityCertificate(certConfig, CertIdentity, signerCert, signerKey, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Identity Certificate: " + err.Error())
			} else {
				fmt.Println("Generating Identity Certificate to " + outCert + ", " + outKey + " was successful.")
			}
		}

		identityCert, err := os.ReadFile(outCert)
		if err != nil {
			fmt.Println("Unable to read Identity Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Certificate from " + outCert + " was successful.")
			IdentityCert = identityCert
		}
		identityKey, err := os.ReadFile(outKey)
		if err != nil {
			fmt.Println("Unable to read Identity Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Private Key from " + outKey + " was successful.")
			IdentityKey = identityKey
		}
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func generateRootCA(certConfig generateCertificate.Configuration, outCert, outKey string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	cert, err := generateCertificate.GenerateRootCA(certConfig, priv)
	if err != nil {
		return err
	}
	err = WriteCertOut(outCert, cert)
	if err != nil {
		return err
	}
	err = WritePrivateKey(outKey, priv)
	if err != nil {
		return err
	}
	return nil
}

func generateIntermediateCertificate(certConfig generateCertificate.Configuration, signCert, signKey, outCert, outKey string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	signerCert, err := security.LoadX509(signCert)
	if err != nil {
		return err
	}
	signerKey, err := security.LoadX509PrivateKey(signKey)
	if err != nil {
		return err
	}
	cert, err := generateCertificate.GenerateIntermediateCA(certConfig, priv, signerCert, signerKey)
	if err != nil {
		return err
	}
	err = WriteCertOut(outCert, cert)
	if err != nil {
		return err
	}
	err = WritePrivateKey(outKey, priv)
	if err != nil {
		return err
	}
	return nil
}

func generateIdentityCertificate(certConfig generateCertificate.Configuration, identity, signCert, signKey, outCert, outKey string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	signerCert, err := security.LoadX509(signCert)
	if err != nil {
		return err
	}
	signerKey, err := security.LoadX509PrivateKey(signKey)
	if err != nil {
		return err
	}
	cert, err := generateCertificate.GenerateIdentityCert(certConfig, identity, priv, signerCert, signerKey)
	if err != nil {
		return err
	}
	err = WriteCertOut(outCert, cert)
	if err != nil {
		return err
	}
	err = WritePrivateKey(outKey, priv)
	if err != nil {
		return err
	}
	return nil
}

func WriteCertOut(filename string, cert []byte) error {
	certOut, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", filename, err)
	}
	_, err = certOut.Write(cert)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("error closing %s: %w", filename, err)
	}
	return nil
}

func WritePrivateKey(filename string, priv *ecdsa.PrivateKey) error {
	privBlock, err := pemBlockForKey(priv)
	if err != nil {
		return fmt.Errorf("failed to encode priv key %s for writing: %w", filename, err)
	}

	keyOut, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", filename, err)
	}

	if err := pem.Encode(keyOut, privBlock); err != nil {
		return fmt.Errorf("failed to write data to %s: %w", filename, err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("error closing %s: %w", filename, err)
	}
	return nil
}

func pemBlockForKey(k *ecdsa.PrivateKey) (*pem.Block, error) {
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil, err
	}
	return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
}

func NewSecureClient() (*local.Client, error) {
	fmt.Println(len(MfgTrustedCA))
	setupSecureClient := SetupSecureClient{}
	if len(MfgTrustedCA) > 0 {
		mfgTrustedCABlock, _ := pem.Decode(MfgTrustedCA)
		if mfgTrustedCABlock == nil {
			return nil, fmt.Errorf("mfgTrustedCABlock is empty")
		}
		mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
		if err != nil {
			return nil, err
		}
		setupSecureClient.mfgCA = mfgCA
	}

	if len(MfgCert) > 0 && len(MfgKey) > 0 {
		mfgCert, err := tls.X509KeyPair(MfgCert, MfgKey)
		if err != nil {
			return nil, fmt.Errorf("cannot X509KeyPair: %w", err)
		}
		setupSecureClient.mfgCert = mfgCert
	}

	if len(IdentityTrustedCA) > 0 {
		identityTrustedCABlock, _ := pem.Decode(IdentityTrustedCA)
		if identityTrustedCABlock == nil {
			return nil, fmt.Errorf("identityTrustedCABlock is empty")
		}
		identityTrustedCACert, err := x509.ParseCertificates(identityTrustedCABlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse cert: %w", err)
		}
		setupSecureClient.ca = identityTrustedCACert
	}
	var cfg local.Config
	if len(IdentityIntermediateCA) > 0 && len(IdentityIntermediateCAKey) > 0 {
		cfg = local.Config{
			// DisablePeerTCPSignalMessageCSMs: true,
			DeviceOwnershipSDK: &local.DeviceOwnershipSDKConfig{
				ID:      CertIdentity,
				Cert:    string(IdentityIntermediateCA),
				CertKey: string(IdentityIntermediateCAKey),
				CreateSignerFunc: func(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore time.Time, validNotAfter time.Time) core.CertificateSigner {
					return signer.NewOCFIdentityCertificate(caCert, caKey, validNotBefore, validNotAfter)
				},
			},
		}
	} else {
		cfg = local.Config{
			// DisablePeerTCPSignalMessageCSMs: true,
			DeviceCacheExpirationSeconds: 3600,
			DeviceOwnershipSDK: &local.DeviceOwnershipSDKConfig{
				ID: CertIdentity,
			},
		}
	}

	client, err := local.NewClientFromConfig(&cfg, &setupSecureClient, nil)

	if err != nil {
		return nil, err
	}
	err = client.Initialization(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}
