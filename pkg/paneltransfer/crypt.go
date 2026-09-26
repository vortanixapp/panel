package paneltransfer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	sealedMagic = "VXTRANSFER1"
	saltSize    = 32
	prefixSize  = 4
	plainChunk  = 1 << 20
	scryptN     = 1 << 16
	scryptR     = 8
	scryptP     = 1
	maxHead     = 64 << 10
)

var (
	ErrPassword = errors.New("неверный пароль архива")
	ErrArchive  = errors.New("архив повреждён или обрезан")
)

type Head struct {
	PanelVersion string `json:"panel_version"`
	CreatedAt    string `json:"created_at"`
	Encrypted    bool   `json:"encrypted"`
}

func derive(password string, salt []byte) (cipher.AEAD, error) {
	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func nonce(prefix []byte, counter uint64) []byte {
	out := make([]byte, 12)
	copy(out, prefix)
	binary.BigEndian.PutUint64(out[prefixSize:], counter)
	return out
}

func aad(counter uint64, last bool) []byte {
	out := make([]byte, 9)
	binary.BigEndian.PutUint64(out, counter)
	if last {
		out[8] = 1
	}
	return out
}

func Seal(w io.Writer, password string, head Head) (io.WriteCloser, error) {
	if password == "" {
		return nil, errors.New("укажите пароль для архива")
	}
	head.Encrypted = true
	raw, err := json.Marshal(head)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, saltSize)
	prefix := make([]byte, prefixSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(prefix); err != nil {
		return nil, err
	}
	aead, err := derive(password, salt)
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(w, sealedMagic); err != nil {
		return nil, err
	}
	if err := writeUint32(w, uint32(len(raw))); err != nil {
		return nil, err
	}
	if _, err := w.Write(raw); err != nil {
		return nil, err
	}
	if _, err := w.Write(salt); err != nil {
		return nil, err
	}
	if _, err := w.Write(prefix); err != nil {
		return nil, err
	}
	return &sealWriter{w: w, aead: aead, prefix: prefix, buf: make([]byte, 0, plainChunk)}, nil
}

type sealWriter struct {
	w      io.Writer
	aead   cipher.AEAD
	prefix []byte
	buf    []byte
	count  uint64
	closed bool
}

func (s *sealWriter) Write(b []byte) (int, error) {
	if s.closed {
		return 0, errors.New("архив уже закрыт")
	}
	written := 0
	for len(b) > 0 {
		room := plainChunk - len(s.buf)
		n := len(b)
		if n > room {
			n = room
		}
		s.buf = append(s.buf, b[:n]...)
		b = b[n:]
		written += n
		if len(s.buf) == plainChunk {
			if err := s.flush(false); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func (s *sealWriter) flush(last bool) error {
	sealed := s.aead.Seal(nil, nonce(s.prefix, s.count), s.buf, aad(s.count, last))
	flag := byte(0)
	if last {
		flag = 1
	}
	if err := writeByte(s.w, flag); err != nil {
		return err
	}
	if err := writeUint32(s.w, uint32(len(sealed))); err != nil {
		return err
	}
	if _, err := s.w.Write(sealed); err != nil {
		return err
	}
	s.buf = s.buf[:0]
	s.count++
	return nil
}

func (s *sealWriter) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.flush(true)
}

func ReadHead(r io.Reader) (*Head, error) {
	magic := make([]byte, len(sealedMagic))
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, ErrArchive
	}
	if string(magic) != sealedMagic {
		return nil, errors.New("это не архив переноса панели")
	}
	size, err := readUint32(r)
	if err != nil || size == 0 || size > maxHead {
		return nil, ErrArchive
	}
	raw := make([]byte, size)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, ErrArchive
	}
	var head Head
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, ErrArchive
	}
	return &head, nil
}

func OpenAfterHead(r io.Reader, password string) (io.Reader, error) {
	if password == "" {
		return nil, errors.New("укажите пароль архива")
	}
	salt := make([]byte, saltSize)
	prefix := make([]byte, prefixSize)
	if _, err := io.ReadFull(r, salt); err != nil {
		return nil, ErrArchive
	}
	if _, err := io.ReadFull(r, prefix); err != nil {
		return nil, ErrArchive
	}
	aead, err := derive(password, salt)
	if err != nil {
		return nil, err
	}
	return &openReader{r: r, aead: aead, prefix: prefix}, nil
}

func Open(r io.Reader, password string) (*Head, io.Reader, error) {
	head, err := ReadHead(r)
	if err != nil {
		return nil, nil, err
	}
	body, err := OpenAfterHead(r, password)
	if err != nil {
		return head, nil, err
	}
	return head, body, nil
}

type openReader struct {
	r      io.Reader
	aead   cipher.AEAD
	prefix []byte
	buf    []byte
	count  uint64
	done   bool
	first  bool
}

func (o *openReader) Read(b []byte) (int, error) {
	for len(o.buf) == 0 {
		if o.done {
			return 0, io.EOF
		}
		if err := o.next(); err != nil {
			return 0, err
		}
	}
	n := copy(b, o.buf)
	o.buf = o.buf[n:]
	return n, nil
}

func (o *openReader) next() error {
	flag, err := readByte(o.r)
	if err != nil {
		return ErrArchive
	}
	if flag > 1 {
		return ErrArchive
	}
	size, err := readUint32(o.r)
	if err != nil || int(size) < o.aead.Overhead() || size > plainChunk+uint32(o.aead.Overhead()) {
		return ErrArchive
	}
	sealed := make([]byte, size)
	if _, err := io.ReadFull(o.r, sealed); err != nil {
		return ErrArchive
	}
	plain, err := o.aead.Open(nil, nonce(o.prefix, o.count), sealed, aad(o.count, flag == 1))
	if err != nil {
		if !o.first {
			return ErrPassword
		}
		return ErrArchive
	}
	o.first = true
	o.count++
	o.buf = plain
	if flag == 1 {
		o.done = true
	}
	return nil
}
