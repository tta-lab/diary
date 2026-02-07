# Implementation Notes

## Age Encryption Reference (from chezmoi)

Reference: `/Users/neil/Code/guion-opensource/chezmoi/internal/chezmoi/ageencryption.go`

### Key Patterns

#### 1. Encryption Pattern

```go
func encrypt(plaintext []byte, recipients []age.Recipient) ([]byte, error) {
    ciphertextBuffer := &bytes.Buffer{}
    armoredWriter := armor.NewWriter(ciphertextBuffer)

    encryptWriter, err := age.Encrypt(armoredWriter, recipients...)
    if err != nil {
        return nil, err
    }

    if _, err := io.Copy(encryptWriter, bytes.NewReader(plaintext)); err != nil {
        return nil, err
    }

    if err := encryptWriter.Close(); err != nil {
        return nil, err
    }

    if err := armoredWriter.Close(); err != nil {
        return nil, err
    }

    return ciphertextBuffer.Bytes(), nil
}
```

**Steps:**
1. Create buffer for ciphertext output
2. Wrap with `armor.NewWriter` (for ASCII-armored output)
3. Call `age.Encrypt(armoredWriter, recipients...)`
4. Copy plaintext into encrypt writer
5. **Important:** Close encrypt writer first
6. **Important:** Close armored writer second
7. Return buffer bytes

#### 2. Decryption Pattern

```go
func decrypt(ciphertext []byte, identities []age.Identity) ([]byte, error) {
    var ciphertextReader io.Reader = bytes.NewReader(ciphertext)

    // Check if armored
    if bytes.HasPrefix(ciphertext, []byte(armor.Header)) {
        ciphertextReader = armor.NewReader(ciphertextReader)
    }

    plaintextReader, err := age.Decrypt(ciphertextReader, identities...)
    if err != nil {
        return nil, err
    }

    plaintextBuffer := &bytes.Buffer{}
    if _, err := io.Copy(plaintextBuffer, plaintextReader); err != nil {
        return nil, err
    }

    return plaintextBuffer.Bytes(), nil
}
```

**Steps:**
1. Create reader from ciphertext bytes
2. Check if armored (starts with `-----BEGIN AGE ENCRYPTED FILE-----`)
3. If armored, wrap with `armor.NewReader`
4. Call `age.Decrypt(ciphertextReader, identities...)`
5. Copy from decrypt reader to buffer
6. Return plaintext bytes

#### 3. Loading Identity (Private Key)

```go
func loadIdentity(identityPath string) ([]age.Identity, error) {
    file, err := os.Open(identityPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    return age.ParseIdentities(file)
}
```

**Identity file format:**
```
# created: 2026-02-07T12:00:00Z
# public key: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
AGE-SECRET-KEY-1XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

#### 4. Loading Recipient (Public Key)

```go
func parseRecipient(recipientString string) (age.Recipient, error) {
    return age.ParseX25519Recipient(recipientString)
}
```

**Recipient format:**
```
age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Or extract from identity:
```go
identity, _ := age.GenerateX25519Identity()
recipient := identity.Recipient()
```

#### 5. Key Generation

```go
func generateKey() (age.Identity, age.Recipient, error) {
    identity, err := age.GenerateX25519Identity()
    if err != nil {
        return nil, nil, err
    }

    recipient := identity.Recipient()
    return identity, recipient, nil
}
```

**Output format:**
```go
fmt.Printf("# created: %s\n", time.Now().Format(time.RFC3339))
fmt.Printf("# public key: %s\n", recipient)
fmt.Printf("%s\n", identity)
```

### Implementation Plan for diary CLI

#### Phase 1: Core Encryption

**File:** `internal/crypto/age.go`

1. **Implement `LoadIdentity(keyPath string)`:**
   - Open key file at path
   - Use `age.ParseIdentities(file)`
   - Return first identity (we only use one key per user)

2. **Implement `Encrypt(plaintext []byte, recipientPublicKey string)`:**
   - Parse recipient with `age.ParseX25519Recipient`
   - Follow encryption pattern above
   - Return armored ciphertext

3. **Implement `Decrypt(ciphertext []byte, identityKeyPath string)`:**
   - Load identity from key path
   - Follow decryption pattern above
   - Return plaintext

4. **Implement `GenerateKey(w io.Writer)`:**
   - Use `age.GenerateX25519Identity()`
   - Write formatted output (created timestamp, public key, identity)

#### Usage in diary CLI

```go
// For encryption (when appending)
keyPath := "~/.config/diary/neil.age.key"
identity, _ := crypto.LoadIdentity(keyPath)
recipient := identity.Recipient().String()
ciphertext, _ := crypto.Encrypt(plaintext, recipient)

// For decryption (when reading)
keyPath := "~/.config/diary/neil.age.key"
plaintext, _ := crypto.Decrypt(ciphertext, keyPath)
```

### Dependencies

Add to `go.mod`:
```bash
go get filippo.io/age
go get filippo.io/age/armor
```

### Testing Strategy

1. **Unit tests for crypto package:**
   - Test encrypt → decrypt → verify plaintext matches
   - Test with different key sizes
   - Test armored vs binary
   - Test error cases (wrong key, corrupted data)

2. **Integration tests:**
   - Generate key
   - Encrypt diary entry
   - Decrypt diary entry
   - Verify content matches

3. **Reference tests:**
   - Encrypt with diary CLI
   - Decrypt with age CLI (verify compatibility)
   - Encrypt with age CLI
   - Decrypt with diary CLI (verify compatibility)

### Security Considerations

1. **Key permissions:**
   - Identity files should be mode 0600
   - Warn if key file is world-readable

2. **Memory:**
   - Plaintext only in buffers (never written to disk during edit)
   - Use `defer` to ensure cleanup

3. **Error handling:**
   - Don't leak key material in error messages
   - Clear sensitive data from memory on error

### Common Pitfalls (from chezmoi)

1. **Must close writers in order:**
   - Close `encryptWriter` first
   - Then close `armoredWriter`
   - Wrong order → corrupt ciphertext

2. **Check for armored format:**
   - Ciphertext may or may not be armored
   - Check for `armor.Header` prefix before decrypting

3. **Identity vs Recipient:**
   - Identity = private key (for decryption)
   - Recipient = public key (for encryption)
   - Can derive recipient from identity, not vice versa

4. **Multiple identities:**
   - `age.ParseIdentities()` returns `[]age.Identity`
   - File can contain multiple keys
   - For diary CLI, we use first identity only

## Next Steps

1. Add age dependencies to go.mod
2. Implement `internal/crypto/age.go` following patterns above
3. Write unit tests
4. Test encrypt/decrypt round-trip
5. Move to Phase 2: implement read/append commands
