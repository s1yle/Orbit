package cmd

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
)

// EncryptBackup encrypts the backup data using hybrid encryption
// Returns the encrypted symmetric key and encrypted data
func EncryptBackup(backupData []byte, publicKey *rsa.PublicKey) ([]byte, []byte, error) {
	// Generate a random symmetric key (AES-256)
	symmetricKey := make([]byte, 32) // 256 bits
	if _, err := rand.Read(symmetricKey); err != nil {
		return nil, nil, fmt.Errorf("failed to generate symmetric key: %v", err)
	}

	// Encrypt the backup data with AES-GCM
	encryptedData, err := encryptWithAES(symmetricKey, backupData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt data with AES: %v", err)
	}

	// Encrypt the symmetric key with RSA-OAEP
	encryptedSymmetricKey, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		publicKey,
		symmetricKey,
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt symmetric key: %v", err)
	}

	return encryptedSymmetricKey, encryptedData, nil
}

// encryptWithAES encrypts data using AES-GCM
func encryptWithAES(key []byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// LoadPublicKey loads an RSA public key from a PEM file
func LoadPublicKey(publicKeyPath string) (*rsa.PublicKey, error) {
	// Read the public key file
	pemData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %v", err)
	}

	// Decode PEM block
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	// Parse the public key
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try parsing as PKCS1
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %v", err)
		}
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("loaded key is not an RSA public key")
	}

	return rsaPub, nil
}

// CreateEncryptedOrbitFile creates an encrypted .orbit file with proper structure
func CreateEncryptedOrbitFile(encryptedSymmetricKey, encryptedData []byte) error {
	backupFile, err := os.Create("backup.orbit")
	if err != nil {
		return err
	}
	defer backupFile.Close()

	// Write file header to identify encrypted format
	header := []byte("ORBIT_ENCRYPTED_v1.0\n")
	if _, err := backupFile.Write(header); err != nil {
		return err
	}

	// Write the encrypted symmetric key length (4 bytes)
	keyLen := make([]byte, 4)
	keyLen[0] = byte(len(encryptedSymmetricKey) >> 24)
	keyLen[1] = byte(len(encryptedSymmetricKey) >> 16)
	keyLen[2] = byte(len(encryptedSymmetricKey) >> 8)
	keyLen[3] = byte(len(encryptedSymmetricKey))
	if _, err := backupFile.Write(keyLen); err != nil {
		return err
	}

	// Write the encrypted symmetric key
	if _, err := backupFile.Write(encryptedSymmetricKey); err != nil {
		return err
	}

	// Write the encrypted data
	if _, err := backupFile.Write(encryptedData); err != nil {
		return err
	}

	return nil
}
