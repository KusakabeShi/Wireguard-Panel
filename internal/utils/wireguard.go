package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

func GenerateWGPrivateKey() (string, error) {
	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", fmt.Errorf("failed to generate random data for private key:-> %v", err)
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

func PrivToPublic(privateKeyB64 string) (string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 private key:-> %w", err)
	}
	if len(privateKeyBytes) != 32 {
		return "", fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKeyBytes))
	}
	var privateKey [32]byte
	copy(privateKey[:], privateKeyBytes)
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return base64.StdEncoding.EncodeToString(publicKey[:]), nil
}

func GenerateWGKeyPair() (privateKey, publicKey string, err error) {
	privateKey, err = GenerateWGPrivateKey()
	if err != nil {
		return "", "", err
	}

	publicKey, err = PrivToPublic(privateKey)
	if err != nil {
		return "", "", err
	}

	return privateKey, publicKey, nil
}
