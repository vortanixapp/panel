package paneltransfer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	plainMagic  = "VXPLAIN1"
	maxPartName = 255
	maxChunk    = 4 << 20

	PartManifest = "manifest.json"
	PartEnv      = "env.json"
	PartDB       = "db.dump"
	PartUploads  = "uploads.tar.gz"
)

var ErrFormat = errors.New("поток переноса повреждён")

type Writer struct {
	w      io.Writer
	header bool
	part   *partWriter
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (t *Writer) Part(name string) (io.Writer, error) {
	if name == "" || len(name) > maxPartName {
		return nil, fmt.Errorf("недопустимое имя части: %q", name)
	}
	if err := t.closePart(); err != nil {
		return nil, err
	}
	if !t.header {
		if _, err := io.WriteString(t.w, plainMagic); err != nil {
			return nil, err
		}
		t.header = true
	}
	if err := writeByte(t.w, 1); err != nil {
		return nil, err
	}
	if err := writeByte(t.w, byte(len(name))); err != nil {
		return nil, err
	}
	if _, err := io.WriteString(t.w, name); err != nil {
		return nil, err
	}
	t.part = &partWriter{w: t.w}
	return t.part, nil
}

func (t *Writer) closePart() error {
	if t.part == nil {
		return nil
	}
	p := t.part
	t.part = nil
	return p.finish()
}

func (t *Writer) Close() error {
	if err := t.closePart(); err != nil {
		return err
	}
	if !t.header {
		if _, err := io.WriteString(t.w, plainMagic); err != nil {
			return err
		}
		t.header = true
	}
	return writeByte(t.w, 0)
}

type partWriter struct {
	w    io.Writer
	done bool
}

func (p *partWriter) Write(b []byte) (int, error) {
	if p.done {
		return 0, errors.New("часть уже закрыта")
	}
	written := 0
	for len(b) > 0 {
		n := len(b)
		if n > maxChunk {
			n = maxChunk
		}
		if err := writeUint32(p.w, uint32(n)); err != nil {
			return written, err
		}
		if _, err := p.w.Write(b[:n]); err != nil {
			return written, err
		}
		written += n
		b = b[n:]
	}
	return written, nil
}

func (p *partWriter) finish() error {
	if p.done {
		return nil
	}
	p.done = true
	return writeUint32(p.w, 0)
}

type Reader struct {
	r      io.Reader
	header bool
	part   *partReader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

func (t *Reader) Next() (string, io.Reader, error) {
	if t.part != nil {
		if err := t.part.drain(); err != nil {
			return "", nil, err
		}
		t.part = nil
	}
	if !t.header {
		magic := make([]byte, len(plainMagic))
		if _, err := io.ReadFull(t.r, magic); err != nil {
			return "", nil, ErrFormat
		}
		if string(magic) != plainMagic {
			return "", nil, ErrFormat
		}
		t.header = true
	}
	kind, err := readByte(t.r)
	if err != nil {
		return "", nil, ErrFormat
	}
	switch kind {
	case 0:
		return "", nil, io.EOF
	case 1:
	default:
		return "", nil, ErrFormat
	}
	size, err := readByte(t.r)
	if err != nil || size == 0 {
		return "", nil, ErrFormat
	}
	name := make([]byte, size)
	if _, err := io.ReadFull(t.r, name); err != nil {
		return "", nil, ErrFormat
	}
	t.part = &partReader{r: t.r}
	return string(name), t.part, nil
}

type partReader struct {
	r    io.Reader
	left int
	done bool
}

func (p *partReader) Read(b []byte) (int, error) {
	if p.left == 0 {
		if p.done {
			return 0, io.EOF
		}
		size, err := readUint32(p.r)
		if err != nil {
			return 0, ErrFormat
		}
		if size == 0 {
			p.done = true
			return 0, io.EOF
		}
		if size > maxChunk {
			return 0, ErrFormat
		}
		p.left = int(size)
	}
	if len(b) > p.left {
		b = b[:p.left]
	}
	n, err := p.r.Read(b)
	p.left -= n
	if err == io.EOF && (n > 0 || p.left > 0) {
		err = ErrFormat
	}
	return n, err
}

func (p *partReader) drain() error {
	_, err := io.Copy(io.Discard, p)
	return err
}

func writeByte(w io.Writer, b byte) error {
	_, err := w.Write([]byte{b})
	return err
}

func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func writeUint32(w io.Writer, v uint32) error {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	_, err := w.Write(b[:])
	return err
}

func readUint32(r io.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b[:]), nil
}
