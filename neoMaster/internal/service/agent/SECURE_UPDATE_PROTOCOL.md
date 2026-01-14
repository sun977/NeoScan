# Secure Rule Update Protocol (安全规则更新协议)

## 1. 概述
为了防止规则文件在传输过程中被篡改或窃取，Master 与 Agent 之间采用 **加密 + 签名** 的双重保护机制传输规则快照。

- **Confidentiality (机密性)**: 使用 AES-256-GCM 加密内容。
- **Integrity (完整性)**: 使用 HMAC-SHA256 对密文进行签名。

## 2. 密钥管理
- **Shared Secret**: Master 和 Agent 共享一个密钥字符串 (`security.agent.rule_encryption_key`)。
- **Encryption Key**: `SHA256(Shared Secret)` (32 bytes)。
- **Signature Key**: `Shared Secret` (Raw string)。

## 3. 传输协议

### 请求
Agent 发起 HTTP GET 请求获取规则快照：
```http
GET /api/v1/agent/rules/fingerprint/download
```

### 响应
Master 返回加密后的数据流，并在 Header 中包含签名和加密标识。

**Headers**:
- `Content-Type`: `application/octet-stream`
- `X-Rule-Signature`: `<HMAC-SHA256 Hex String>` (针对 Response Body 的签名)
- `X-Content-Encryption`: `aes-gcm`

**Body**:
- Binary Data: `Nonce (12 bytes) + Ciphertext + Tag (16 bytes)`

## 4. Agent 处理流程 (伪代码)

Agent 收到响应后，必须严格按照以下顺序处理：

1.  **Verify Signature (验证签名)**:
    *   读取 Response Body 为 `encryptedBytes`。
    *   读取 Header `X-Rule-Signature` 为 `remoteSignature`。
    *   计算本地签名: `localSignature = HMAC-SHA256(SharedSecret, encryptedBytes)`。
    *   对比 `localSignature` 与 `remoteSignature`。
    *   **如果签名不匹配，立即丢弃数据并报错 (防止篡改)。**

2.  **Decrypt Content (解密内容)**:
    *   检查 Header `X-Content-Encryption` 是否为 `aes-gcm`。
    *   生成解密密钥: `key = SHA256(SharedSecret)`。
    *   分离 Nonce 和 Ciphertext:
        *   `nonce = encryptedBytes[:12]`
        *   `ciphertext = encryptedBytes[12:]`
    *   使用 AES-GCM 解密: `plaintext = AES_GCM_Open(key, nonce, ciphertext)`。
    *   **如果解密失败 (Tag 校验失败)，报错 (防止篡改)。**

3.  **Use Content (使用内容)**:
    *   `plaintext` 即为原始的 ZIP 文件数据。
    *   解压 ZIP 并加载规则。

## 5. Golang 实现示例

```go
func DecryptRule(secret string, encryptedData []byte) ([]byte, error) {
    // 1. Derive Key
    key := sha256.Sum256([]byte(secret))

    // 2. Create Cipher
    block, err := aes.NewCipher(key[:])
    if err != nil {
        return nil, err
    }

    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    // 3. Validate Length
    nonceSize := aesgcm.NonceSize() // 12
    if len(encryptedData) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    // 4. Split Nonce
    nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

    // 5. Decrypt
    plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }

    return plaintext, nil
}
```
