package agent

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"neomaster/internal/config"

	"github.com/stretchr/testify/assert"
)

// decryptData 使用 AES-GCM 解密数据 (用于测试验证)
func decryptData(key string, ciphertext []byte) ([]byte, error) {
	k := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedMessage := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesgcm.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func TestGetEncryptedSnapshot(t *testing.T) {
	// 1. Setup Environment
	tmpDir, err := ioutil.TempDir("", "neoscan_rules_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ruleDir := filepath.Join(tmpDir, "fingerprint")
	if err := os.MkdirAll(ruleDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFileContent := []byte(`{"test": "content"}`)
	if err := ioutil.WriteFile(filepath.Join(ruleDir, "test.json"), testFileContent, 0644); err != nil {
		t.Fatal(err)
	}

	secretKey := "test-secret-key-123456"

	cfg := &config.Config{
		App: config.AppConfig{
			Rules: config.RulesConfig{
				Fingerprint: config.RuleDirConfig{
					Dir: "fingerprint",
				},
				RootPath: tmpDir,
			},
		},
		Security: config.SecurityConfig{
			Agent: config.AgentConfig{
				RuleEncryptionKey: secretKey,
			},
		},
	}

	svc := NewAgentUpdateService(cfg)

	// 2. Execute
	snapshot, err := svc.GetEncryptedSnapshot(context.Background(), RuleTypeFingerprint)
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)

	// 3. Verify Signature
	assert.NotEmpty(t, snapshot.Signature)
	
	// Verify signature matches the encrypted content
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(snapshot.Bytes)
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	assert.Equal(t, expectedSignature, snapshot.Signature)

	// 4. Verify Encryption
	assert.Equal(t, "application/octet-stream", snapshot.ContentType)
	
	// Try to decrypt
	decryptedZip, err := decryptData(secretKey, snapshot.Bytes)
	assert.NoError(t, err, "Decryption failed")
	
	// Verify decrypted content is a valid ZIP (starts with PK header)
	// ZIP file signature is "PK\x03\x04" (0x50 0x4B 0x03 0x04)
	assert.True(t, len(decryptedZip) > 4)
	assert.Equal(t, []byte{0x50, 0x4B, 0x03, 0x04}, decryptedZip[:4], "Decrypted content is not a valid ZIP")
}
