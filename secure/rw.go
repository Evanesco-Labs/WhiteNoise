package secure

import (
	"encoding/binary"
	pool "github.com/libp2p/go-buffer-pool"
	"io"

	"golang.org/x/crypto/poly1305"
)

const MaxTransportMsgLength = 0xffff
const MaxPlaintextLength = MaxTransportMsgLength - poly1305.TagSize
const LengthPrefixLength = 2

func (s *SecureSession) Read(buf []byte) (int, error) {
	s.readLock.Lock()
	defer s.readLock.Unlock()

	if s.qbuf != nil {
		copied := copy(buf, s.qbuf[s.qseek:])
		s.qseek += copied
		if s.qseek == len(s.qbuf) {
			pool.Put(s.qbuf)
			s.qseek, s.qbuf = 0, nil
		}
		return copied, nil
	}

	nextMsgLen, err := s.readNextInsecureMsgLen()
	if err != nil {
		return 0, err
	}

	if len(buf) >= nextMsgLen {
		if err := s.readNextMsgInsecure(buf[:nextMsgLen]); err != nil {
			return 0, err
		}

		dbuf, err := s.decrypt(buf[:0], buf[:nextMsgLen])
		if err != nil {
			return 0, err
		}

		return len(dbuf), nil
	}

	cbuf := pool.Get(nextMsgLen)
	if err := s.readNextMsgInsecure(cbuf); err != nil {
		return 0, err
	}

	if s.qbuf, err = s.decrypt(cbuf[:0], cbuf); err != nil {
		return 0, err
	}

	s.qseek = copy(buf, s.qbuf)

	return s.qseek, nil
}

func (s *SecureSession) Write(data []byte) (int, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	var (
		written int
		cbuf    []byte
		total   = len(data)
	)

	if total < MaxPlaintextLength {
		cbuf = pool.Get(total + poly1305.TagSize + LengthPrefixLength)
	} else {
		cbuf = pool.Get(MaxTransportMsgLength + LengthPrefixLength)
	}

	defer pool.Put(cbuf)

	for written < total {
		end := written + MaxPlaintextLength
		if end > total {
			end = total
		}

		b, err := s.encrypt(cbuf[:LengthPrefixLength], data[written:end])
		if err != nil {
			return 0, err
		}

		binary.BigEndian.PutUint16(b, uint16(len(b)-LengthPrefixLength))

		_, err = s.writeMsgInsecure(b)
		if err != nil {
			return written, err
		}
		written = end
	}
	return written, nil
}

func (s *SecureSession) readNextInsecureMsgLen() (int, error) {
	_, err := io.ReadFull(s.insecure, s.rlen[:])
	if err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint16(s.rlen[:])), err
}

func (s *SecureSession) readNextMsgInsecure(buf []byte) error {
	_, err := io.ReadFull(s.insecure, buf)
	return err
}

func (s *SecureSession) writeMsgInsecure(data []byte) (int, error) {
	return s.insecure.Write(data)
}
