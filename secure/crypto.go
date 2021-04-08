package secure

import (
	"errors"
)

func (s *SecureSession) encrypt(out, plaintext []byte) ([]byte, error) {
	if s.enc == nil {
		return nil, errors.New("cannot encrypt, no handshake or handshake failed")
	}
	return s.enc.Encrypt(out, nil, plaintext), nil
}

func (s *SecureSession) decrypt(out, ciphertext []byte) ([]byte, error) {
	if s.dec == nil {
		return nil, errors.New("cannot decrypt, no handshake or handshake failed")
	}
	return s.dec.Decrypt(out, nil, ciphertext)
}
