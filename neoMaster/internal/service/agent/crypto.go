package agent

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// encryptData 使用 AES-GCM 加密数据
// key: 原始密钥 (会自动进行 SHA256 处理以符合 AES-256 长度要求)
// plaintext: 待加密数据
// 返回: nonce + ciphertext + tag (GCM 自动附加 tag)
func encryptData(key string, plaintext []byte) ([]byte, error) {
	// 1. 处理密钥，确保长度为 32 字节 (AES-256)
	k := sha256.Sum256([]byte(key))

	// 2. 创建 Cipher Block
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	// 3. 创建 GCM 模式
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 4. 生成 Nonce
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 5. 加密 (Seal appends the result to the first argument, which is the prefix)
	// Output format: Nonce + Ciphertext + Tag
	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}
