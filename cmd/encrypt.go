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
	header := []byte(EncryptedVerStr)
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

// LoadPrivateKey loads an RSA private key from a PEM file
func LoadPrivateKey(privateKeyPath string) (*rsa.PrivateKey, error) {
	// Read the private key file
	pemData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %v", err)
	}

	// Decode PEM block
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try parsing as PKCS8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("loaded key is not an RSA private key")
		}
		return rsaKey, nil
	}

	return privateKey, nil
}

// DecryptBackup decrypts the backup data using private key
func DecryptBackup(encryptedSymmetricKey, encryptedData []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	// Decrypt the symmetric key with RSA-OAEP
	symmetricKey, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		privateKey,
		encryptedSymmetricKey,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt symmetric key: %v", err)
	}

	// Decrypt the backup data with AES-GCM
	decryptedData, err := decryptWithAES(symmetricKey, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data with AES: %v", err)
	}

	return decryptedData, nil
}

// decryptWithAES decrypts data using AES-GCM
func decryptWithAES(key []byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// ReadEncryptedOrbitFile reads and parses an encrypted .orbit file
func ReadEncryptedOrbitFile(orbitFilePath string) ([]byte, []byte, error) {
	fileData, err := os.ReadFile(orbitFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read orbit file: %v", err)
	}

	// Check if it's an encrypted file
	if len(fileData) < len(EncryptedVerStr) ||
		string(fileData[:len(EncryptedVerStr)]) != EncryptedVerStr {
		return nil, nil, errors.New("not an encrypted orbit file")
	}

	// Skip header
	data := fileData[len(EncryptedVerStr):]

	// Read encrypted symmetric key length
	if len(data) < 4 {
		return nil, nil, errors.New("invalid orbit file format: missing key length")
	}
	keyLen := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	data = data[4:]

	// Read encrypted symmetric key
	if len(data) < keyLen {
		return nil, nil, errors.New("invalid orbit file format: key length mismatch")
	}
	encryptedSymmetricKey := data[:keyLen]
	data = data[keyLen:]

	// Remaining data is the encrypted backup
	encryptedData := data

	return encryptedSymmetricKey, encryptedData, nil
}
