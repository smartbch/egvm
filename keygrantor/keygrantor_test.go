package keygrantor

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tyler-smith/go-bip32"
)

func TestDeriveKey(t *testing.T) {
	seed := sha256.Sum256([]byte("abcdefg"))
	rootPriv, err := bip32.NewMasterKey(seed[:])
	fmt.Printf("rootPriv %s\n", rootPriv.B58Serialize())
	rootPub := rootPriv.PublicKey()
	fmt.Printf("rootPub %s\n", rootPub.B58Serialize())
	require.Nil(t, err)
	h := sha256.Sum256([]byte("12345678"))
	derivedPriv := DeriveKey(rootPriv, h)
	s1 := derivedPriv.PublicKey().B58Serialize()
	derivedPub := DeriveKey(rootPub, h)
	s2 := derivedPub.B58Serialize()
	require.Equal(t, s1, s2)
}
