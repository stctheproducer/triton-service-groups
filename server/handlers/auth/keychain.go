package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/account"
	"github.com/joyent/triton-go/authentication"
	"github.com/pkg/errors"
)

// we're using us-west-1 as a miniscule hedge against LDAP latency
const tritonBaseURL = "https://us-west-1.api.joyent.com/"

type Keychain struct {
	*ParsedRequest

	// found  foundKeys
	config *triton.ClientConfig

	AccountKey *account.Key
}

func NewKeychain(req *ParsedRequest) *Keychain {
	signer := &authentication.TestSigner{}
	config := &triton.ClientConfig{
		TritonURL:   tritonBaseURL,
		AccountName: req.AccountName,
		Signers:     []authentication.Signer{signer},
	}

	return &Keychain{
		ParsedRequest: req,
		config:        config,
	}
}

// newClient constructs our Triton AccountClient
func (k *Keychain) newClient() (*account.AccountClient, error) {
	return account.NewClient(k.config)
}

// CheckTriton checks Triton account keys for our TSG key
func (k *Keychain) CheckTriton(ctx context.Context) error {
	a, err := k.newClient()
	if err != nil {
		return errors.Wrap(err, "failed to create account keys client")
	}

	a.SetHeader(k.ParsedRequest.Header())

	listInput := &account.ListKeysInput{}
	keys, err := a.Keys().List(ctx, listInput)
	if err != nil {
		return errors.Wrap(err, "failed to list account keys")
	}
	for _, key := range keys {
		if strings.HasPrefix(key.Name, "tsg-") {
			k.AccountKey = key
			break
		}
	}
	return nil
}

// AddKey adds an account key into Triton, converting the passed in KeyPair into
// a Triton-Go account.Key for use by external consumers.
func (k *Keychain) AddKey(ctx context.Context, keypair *KeyPair) error {
	a, err := k.newClient()
	if err != nil {
		return errors.Wrap(err, "failed to create new key client")
	}

	a.SetHeader(k.ParsedRequest.Header())

	name := fmt.Sprintf("tsg-%s", time.Now().Format("20060102150405"))

	createInput := &account.CreateKeyInput{
		Name: name,
		Key:  keypair.PublicKeyBase64(),
	}
	key, err := a.Keys().Create(ctx, createInput)
	if err != nil {
		return errors.Wrap(err, "failed to create new account key")
	}

	k.AccountKey = key

	return nil
}

func (k *Keychain) HasKey() bool {
	return k.AccountKey != nil
}