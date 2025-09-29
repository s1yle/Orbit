package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type KeyPairs struct {
	PrivateKeyPath string
	PublicKeyPath  string
}

func genKeys() (*KeyPairs, error) {
	var keyPairs KeyPairs

	// Generate a new RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logger.Fatal("Failed to generate key:", err)
		return nil, err
	}

	username, err := getWinUserName()
	username = filepath.Base(username)
	if err != nil {
		logger.Fatal("Failed to get username:", err)
		return nil, err
	}

	// Save the private key
	privateKeyPath := username + "_private_key.pem"
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		logger.Fatal("Failed to create private key file:", err)
		return nil, err
	}
	defer privateKeyFile.Close()

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		logger.Fatal("Failed to write private key:", err)
		return nil, err
	}

	// Save the public key
	publicKeyPath := username + "_public_key.pem"
	publicKeyFile, err := os.Create(publicKeyPath)
	if err != nil {
		logger.Fatal("Failed to create public key file:", err)
		return nil, err
	}
	defer publicKeyFile.Close()

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		logger.Fatal("Failed to marshal public key:", err)
		return nil, err
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		logger.Fatal("Failed to write public key:", err)
		return nil, err
	}

	logger.Infof("Test key pair generated successfully:")
	logger.Infof("- Private key: %s", privateKeyPath)
	logger.Infof("- Public key: %s", publicKeyPath)
	keyPairs.PrivateKeyPath = privateKeyPath
	keyPairs.PublicKeyPath = publicKeyPath
	return &keyPairs, err
}

var generateKeysCmd = &cobra.Command{
	Use:   "generate-keys",
	Short: "Generate a test RSA key pair",
	Long:  `Generates a test RSA key pair and saves them as [user]_private_key.pem and [user]_public_key.pem.`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		keyPairs, err := genKeys()
		if err != nil {
			logger.Fatal("Failed to generate keys:", err)
			return
		}
		logger.Infof("Key pair generated:\nPrivate Key: %s\nPublic Key: %s", keyPairs.PrivateKeyPath, keyPairs.PublicKeyPath)
	},
}

func init() {
	rootCmd.AddCommand(generateKeysCmd)
}
