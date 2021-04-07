package secure

import (
	"errors"
)

func (s *SecureSession) encrypt(out, plaintext []byte) ([]byte, error) {
	if s.enc == nil {
		return nil, errors.New("cannot encrypt, handshake incomplete")
	}
	return s.enc.Encrypt(out, nil, plaintext), nil
}

func (s *SecureSession) decrypt(out, ciphertext []byte) ([]byte, error) {
	if s.dec == nil {
		return nil, errors.New("cannot decrypt, handshake incomplete")
	}
	return s.dec.Decrypt(out, nil, ciphertext)
}
