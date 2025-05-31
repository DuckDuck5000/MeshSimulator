package crypto

import (
	"crypto/rand"
	"errors"
	"golang.org/x/crypto/nacl/box"
)

type KeyPair struct {
	PublicKey  *[32]byte
	PrivateKey *[32]byte
}

// GenerateKeyPair creates a public/private key pair
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &KeyPair{PublicKey: pub, PrivateKey: priv}, nil
}

// EncryptMessage encrypts a message for the recipient
func EncryptMessage(msg []byte, senderPriv, recipientPub *[32]byte) ([]byte, *[24]byte, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, nil, err
	}

	encrypted := box.Seal(nil, msg, &nonce, recipientPub, senderPriv)
	return encrypted, &nonce, nil
}

// DecryptMessage decrypts the received message
func DecryptMessage(cipher []byte, nonce *[24]byte, senderPub, recipientPriv *[32]byte) ([]byte, error) {
	msg, ok := box.Open(nil, cipher, nonce, senderPub, recipientPriv)
	if !ok {
		return nil, errors.New("decryption failed")
	}
	return msg, nil
}
