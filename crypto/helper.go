package crypto

import (
	"errors"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	. "ELAClient/common"
)

func ToAesKey(password []byte) []byte {
	defer ClearBytes(password, len(password))

	first := sha256.Sum256(password)
	second := sha256.Sum256(first[:])

	defer ClearBytes(first[:], 32)

	return second[:]
}

func AesEncrypt(plaintext []byte, key []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("invalid decrypt key")
	}
	blockMode := cipher.NewCBCEncrypter(block, iv)

	ciphertext := make([]byte, len(plaintext))
	blockMode.CryptBlocks(ciphertext, plaintext)

	return ciphertext, nil
}

func AesDecrypt(ciphertext []byte, key []byte, iv []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("invalid decrypt key")
	}

	blockSize := block.BlockSize()

	if len(ciphertext) < blockSize {
		return nil, errors.New("ciphertext too short")
	}

	if len(ciphertext)%blockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	blockModel := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	blockModel.CryptBlocks(plaintext, ciphertext)
	//plaintext = PKCS5UnPadding(plaintext)

	return plaintext, nil
}
