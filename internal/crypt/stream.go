package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

const (
	magic     = "FMDENC1\n"
	saltSize  = 16
	headSize  = len(magic) + saltSize
	chunkSize = 64 << 10
	tagSize   = 16
	sealed    = chunkSize + tagSize
)

var ErrCorrupt = errors.New("crypt: the encrypted data is damaged or the key is wrong")

type ReadSeekCloser interface {
	io.ReadSeekCloser
	io.ReaderAt
	Size() int64
}

func (k *Key) fileAEAD(salt []byte) cipher.AEAD {
	m := hmac.New(sha256.New, k.Sub("files", 32))
	m.Write(salt)
	block, err := aes.NewCipher(m.Sum(nil))
	if err != nil {
		panic(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return aead
}

func nonce(i uint64) []byte {
	n := make([]byte, 12)
	binary.BigEndian.PutUint64(n[4:], i)
	return n
}

func aad(i uint64, last bool) []byte {
	a := make([]byte, 9)
	binary.BigEndian.PutUint64(a, i)
	if last {
		a[8] = 1
	}
	return a
}

type writer struct {
	w    io.Writer
	aead cipher.AEAD
	buf  []byte
	idx  uint64
	err  error
	head bool
	salt []byte
}

func (k *Key) Writer(w io.Writer) io.WriteCloser {
	if k == nil {
		return nopCloser{w}
	}
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	return &writer{w: w, aead: k.fileAEAD(salt), buf: make([]byte, 0, chunkSize), salt: salt}
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func (e *writer) header() error {
	if e.head {
		return nil
	}
	e.head = true
	_, err := e.w.Write(append([]byte(magic), e.salt...))
	return err
}

func (e *writer) Write(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	if e.err = e.header(); e.err != nil {
		return 0, e.err
	}
	n := 0
	for len(p) > 0 {
		if len(e.buf) == chunkSize {
			if e.err = e.flush(false); e.err != nil {
				return n, e.err
			}
		}
		c := min(chunkSize-len(e.buf), len(p))
		e.buf = append(e.buf, p[:c]...)
		p = p[c:]
		n += c
	}
	return n, nil
}

func (e *writer) flush(last bool) error {
	out := e.aead.Seal(nil, nonce(e.idx), e.buf, aad(e.idx, last))
	e.idx++
	e.buf = e.buf[:0]
	_, err := e.w.Write(out)
	return err
}

func (e *writer) Close() error {
	if e.err != nil {
		return e.err
	}
	if e.err = e.header(); e.err != nil {
		return e.err
	}
	e.err = e.flush(true)
	if e.err == nil {
		e.err = os.ErrClosed
		return nil
	}
	return e.err
}

type file struct {
	r      io.ReaderAt
	c      io.Closer
	aead   cipher.AEAD
	size   int64
	chunks int64
	off    int64
	cached int64
	plain  []byte
	raw    []byte
}

func (k *Key) Open(path string) (ReadSeekCloser, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	st, err := fh.Stat()
	if err != nil {
		fh.Close()
		return nil, err
	}
	r, err := k.Reader(fh, st.Size())
	if err != nil {
		fh.Close()
		return nil, err
	}
	if f, ok := r.(*file); ok {
		f.c = fh
		return f, nil
	}
	return plainFile{fh, st.Size()}, nil
}

type plainFile struct {
	*os.File
	size int64
}

func (p plainFile) Size() int64 { return p.size }

func IsSealed(r io.ReaderAt) bool {
	h := make([]byte, len(magic))
	n, _ := r.ReadAt(h, 0)
	return n == len(magic) && string(h) == magic
}

func (k *Key) Reader(r io.ReaderAt, size int64) (io.ReaderAt, error) {
	if !IsSealed(r) {
		return r, nil
	}
	if k == nil {
		return nil, ErrLocked
	}
	salt := make([]byte, saltSize)
	if _, err := r.ReadAt(salt, int64(len(magic))); err != nil {
		return nil, ErrCorrupt
	}
	body := size - int64(headSize)
	chunks := (body + sealed - 1) / sealed
	if chunks < 1 || body-chunks*tagSize < 0 {
		return nil, ErrCorrupt
	}
	return &file{r: r, aead: k.fileAEAD(salt), size: body - chunks*tagSize, chunks: chunks, cached: -1, raw: make([]byte, sealed)}, nil
}

func (f *file) Size() int64 { return f.size }

func (f *file) chunk(i int64) error {
	if i == f.cached {
		return nil
	}
	start := int64(headSize) + i*sealed
	n := int64(sealed)
	if i == f.chunks-1 {
		n = f.size - i*chunkSize + tagSize
	}
	raw := f.raw[:n]
	if _, err := f.r.ReadAt(raw, start); err != nil && !(errors.Is(err, io.EOF) && i == f.chunks-1) {
		return err
	}
	plain, err := f.aead.Open(f.plain[:0], nonce(uint64(i)), raw, aad(uint64(i), i == f.chunks-1))
	if err != nil {
		f.cached = -1
		return ErrCorrupt
	}
	f.plain, f.cached = plain, i
	return nil
}

func (f *file) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("crypt: negative offset")
	}
	n := 0
	for len(p) > 0 {
		if off >= f.size {
			return n, io.EOF
		}
		i := off / chunkSize
		if err := f.chunk(i); err != nil {
			return n, err
		}
		c := copy(p, f.plain[off-i*chunkSize:])
		p, off, n = p[c:], off+int64(c), n+c
	}
	return n, nil
}

func (f *file) Read(p []byte) (int, error) {
	if f.off >= f.size {
		return 0, io.EOF
	}
	if int64(len(p)) > f.size-f.off {
		p = p[:f.size-f.off]
	}
	n, err := f.ReadAt(p, f.off)
	f.off += int64(n)
	if errors.Is(err, io.EOF) && n > 0 {
		err = nil
	}
	return n, err
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += f.off
	case io.SeekEnd:
		offset += f.size
	default:
		return 0, errors.New("crypt: bad whence")
	}
	if offset < 0 {
		return 0, errors.New("crypt: negative position")
	}
	f.off = offset
	return offset, nil
}

func (f *file) Close() error {
	if f.c != nil {
		return f.c.Close()
	}
	return nil
}

func (k *Key) Seal(plain []byte) []byte {
	if k == nil {
		return plain
	}
	var b bytes.Buffer
	w := k.Writer(&b)
	w.Write(plain)
	w.Close()
	return b.Bytes()
}

func (k *Key) Unseal(data []byte) ([]byte, error) {
	r, err := k.Reader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if f, ok := r.(*file); ok {
		out := make([]byte, f.size)
		if _, err := f.ReadAt(out, 0); err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		return out, nil
	}
	return data, nil
}

func (k *Key) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, k.Seal(data), perm)
}

func (k *Key) ReadFile(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return k.Unseal(raw)
}
