package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"sync"

	"promthus/internal/logger"

	"go.uber.org/zap"
)

/*
KMS 接口：约定「设备密钥」相关的三件事：
EncryptDeviceKey：用主密钥加密设备密钥（明文 → 密文，存库）。
DecryptDeviceKey：用主密钥解密设备密钥（密文 → 明文，仅在内存里用）。
ComputeCMAC：用设备密钥对一段数据算 MAC（挑战-应答里算 Response）。
*/
// KMS provides key management operations for device keys.
type KMS interface {
	EncryptDeviceKey(plainKey []byte) ([]byte, error)
	DecryptDeviceKey(encryptedKey []byte) ([]byte, error)
	ComputeCMAC(key, data []byte) ([]byte, error)
}

// LocalKMS：当前实现，把主密钥放在内存里（masterKey），用 mu 保证并发读主密钥时安全（RLock/RUnlock）。
type LocalKMS struct {
	masterKey []byte
	mu        sync.RWMutex
}

var instance KMS
var once sync.Once

func Init(masterKeyPath string) {
	// sync.Once 保证所有线程只执行一次;
	once.Do(func() {
		key, err := os.ReadFile(masterKeyPath)
		if err != nil {
			logger.Warn("master key file not found, generating ephemeral key for development",
				zap.String("path", masterKeyPath))
			// 生成一个临时密钥切片,初始化全0;
			key = make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				logger.Fatal("failed to generate ephemeral master key", zap.Error(err))
			}
		}
		if len(key) < 32 {
			padded := make([]byte, 32)
			copy(padded, key)
			key = padded
		}
		// 把主密钥放在内存里;
		instance = &LocalKMS{masterKey: key[:32]}
	})
}

// instance如果为空,则进行KMS密钥初始化;
func Get() KMS {
	if instance == nil {
		Init("./master.key")
	}
	return instance
}

func (k *LocalKMS) EncryptDeviceKey(plainKey []byte) ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	block, err := aes.NewCipher(k.masterKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return aesGCM.Seal(nonce, nonce, plainKey, nil), nil
}

func (k *LocalKMS) DecryptDeviceKey(encryptedKey []byte) ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	block, err := aes.NewCipher(k.masterKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(encryptedKey) < nonceSize {
		return nil, errors.New("encrypted key too short")
	}

	nonce, ciphertext := encryptedKey[:nonceSize], encryptedKey[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}

// ComputeCMAC computes a MAC over the given data using the device key.
// Uses HMAC-SHA256 truncated to 16 bytes as a portable alternative to AES-CMAC.
// In production with real MCU integration, replace with a proper AES-128-CMAC implementation.
func (k *LocalKMS) ComputeCMAC(key, data []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	full := mac.Sum(nil)
	return full[:16], nil
}
