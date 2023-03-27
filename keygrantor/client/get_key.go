package client

import (
	"encoding/hex"
	"fmt"

	ecies "github.com/ecies/go/v2"
	"github.com/edgelesssys/ego/enclave"
	"github.com/tyler-smith/go-bip32"

	"github.com/smartbch/pureauth/keygrantor"
)

func GetKey(keyGrantorRpc string) (*bip32.Key, error) {
	extPrivKey := keygrantor.GetRandomExtPrivKey()
	privKey := keygrantor.Bip32KeyToEciesKey(keygrantor.NewKeyFromRootKey(extPrivKey))

	data := privKey.PublicKey.Bytes(true)
	reportBz, err := enclave.GetRemoteReport(data)
	if err != nil {
		return nil, fmt.Errorf("failed to get report attestation report: %w", err)
	}
	jwtToken, err := enclave.CreateAzureAttestationToken(data, keygrantor.AttestationProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get azure attestation token: %w", err)
	}
	url := fmt.Sprintf("%s/getkey?report=%stoken=%s", keyGrantorRpc,
		hex.EncodeToString(reportBz), jwtToken)
	encryptKeyBz, err := keygrantor.HttpGet(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	keyBz, err := ecies.Decrypt(privKey, encryptKeyBz)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	key, err := bip32.Deserialize(keyBz)
	if err != nil {
		return nil, fmt.Errorf("failed to Deserialize key: %w", err)
	}

	return key, nil
}
