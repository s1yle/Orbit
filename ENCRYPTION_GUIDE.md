# Orbit Backup Encryption Guide

## Overview

Orbit now supports encrypted backups using public key cryptography. This guide explains how to use this feature.

## How It Works

Orbit uses a hybrid encryption approach:

1. **Symmetric Encryption (AES-256-GCM)**: The actual backup data is encrypted using a randomly generated symmetric key.
2. **Asymmetric Encryption (RSA-OAEP)**: The symmetric key is encrypted using the user's public key.
3. **File Format**: The encrypted backup file contains:
   - A header identifying it as an encrypted Orbit backup
   - The encrypted symmetric key (encrypted with your public key)
   - The encrypted backup data

## Generating Key Pairs

### Using OpenSSL

Generate a private key (keep this secure and never share it):

```bash
openssl genrsa -out private_key.pem 2048
```

Extract the public key from the private key:

```bash
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

### Using Other Tools

You can use any tool that generates RSA key pairs in PEM format. The public key should be in PKCS#1 or PKCS#8 format.

## Using Encryption

### Creating an Encrypted Backup

To create an encrypted backup, use the `-k` or `--public-key` flag with the path to your public key:

```bash
orbit save -k public_key.pem
```

### Without Encryption

If you don't specify a public key, the backup will be created without encryption (legacy behavior):

```bash
orbit save
```

## Decryption (Future Implementation)

To decrypt an encrypted backup, you would need:

1. The encrypted `.orbit` file
2. Your private key (the one that matches the public key used for encryption)

The decryption process would:
1. Read the encrypted symmetric key from the backup file
2. Decrypt the symmetric key using your private key
3. Use the symmetric key to decrypt the backup data
4. Extract the original files

**Note**: Decryption functionality is not yet implemented in Orbit. You would need to use external tools or wait for a future release that includes decryption support.

## Security Considerations

- **Keep your private key secure**: Anyone with your private key can decrypt your backups.
- **Use strong keys**: Use at least 2048-bit RSA keys.
- **Backup your keys**: If you lose your private key, you cannot recover your encrypted backups.
- **Key storage**: Store private keys in a secure location, preferably encrypted with a passphrase.

## File Format Specification

Encrypted Orbit files have the following structure:

```
[Header] "ORBIT_ENCRYPTED_v1.0\n" (19 bytes)
[Key Length] 4 bytes (big-endian uint32)
[Encrypted Symmetric Key] (length specified by Key Length)
[Encrypted Backup Data] (remaining bytes)
```

The encrypted backup data is a zip file encrypted with AES-256-GCM.
