package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	b64 "encoding/base64"
	"encoding/hex"
	"io"
)

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func encrypt(data []byte, passphrase string) ([]byte, error) {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func decrypt(data []byte, passphrase string) ([]byte, error) {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// EncodeAndEncrypt encrypt the string data using passphrase and base64 encode
func EncodeAndEncrypt(s, passphrase string) (string, error) {
	bytes, err := encrypt([]byte(s), passphrase)
	if err != nil {
		return "", err
	}
	uEnc := b64.URLEncoding.EncodeToString(bytes)
	return uEnc, nil
}

// DecodeAndDecrypt decode and decrypt base64 data
func DecodeAndDecrypt(s, passphrase string) (string, error) {
	sDec, err := b64.URLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	bytes, err := decrypt(sDec, passphrase)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
