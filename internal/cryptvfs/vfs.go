package cryptvfs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/crypto/xts"
	"modernc.org/libc"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	PageSize = 4096
	frameHdr = 24
	shimHdr  = 16
)

const (
	kindDB = iota
	kindWAL
	kindJournal
	kindTemp
)

type Keys struct {
	DB, WAL, Journal []byte
}

type state struct {
	kind    int
	sub     uintptr
	methods uintptr
	xts     *xts.Cipher
	temp    cipher.Block
	scratch uintptr
}

type entry struct {
	base uintptr
	db   *xts.Cipher
	wal  *xts.Cipher
	jrn  *xts.Cipher
}

var (
	mu      sync.Mutex
	next    uintptr
	files   = map[uintptr]*state{}
	vfses   = map[uintptr]*entry{}
	methods sqlite3.Tsqlite3_io_methods
	once    sync.Once
)

func fp[T any](f T) uintptr {
	return *(*uintptr)(unsafe.Pointer(&struct{ f T }{f}))
}

func ptr(p uintptr) unsafe.Pointer {
	return unsafe.Add(nil, p)
}

func handle() uintptr {
	next++
	return next
}

func Register(name string, keys Keys) error {
	once.Do(initMethods)
	mu.Lock()
	defer mu.Unlock()
	tls := libc.NewTLS()
	defer tls.Close()
	base := sqlite3.Xsqlite3_vfs_find(tls, 0)
	if base == 0 {
		return fmt.Errorf("cryptvfs: no default VFS")
	}
	e := &entry{base: base}
	var err error
	if e.db, err = xts.NewCipher(aes.NewCipher, keys.DB); err != nil {
		return err
	}
	if e.wal, err = xts.NewCipher(aes.NewCipher, keys.WAL); err != nil {
		return err
	}
	if e.jrn, err = xts.NewCipher(aes.NewCipher, keys.Journal); err != nil {
		return err
	}
	cname, err := libc.CString(name)
	if err != nil {
		return err
	}
	p := libc.Xmalloc(tls, libc.Tsize_t(unsafe.Sizeof(sqlite3.Tsqlite3_vfs{})))
	if p == 0 {
		return fmt.Errorf("cryptvfs: out of memory")
	}
	v := (*sqlite3.Tsqlite3_vfs)(ptr(p))
	*v = *(*sqlite3.Tsqlite3_vfs)(ptr(base))
	h := handle()
	vfses[h] = e
	v.FszOsFile = shimHdr + v.FszOsFile
	v.FpNext = 0
	v.FzName = cname
	v.FxOpen = fp(xOpen)
	app := (*[2]uintptr)(ptr(libc.Xmalloc(tls, 16)))
	app[0], app[1] = (*sqlite3.Tsqlite3_vfs)(ptr(base)).FpAppData, h
	v.FpAppData = uintptr(unsafe.Pointer(app))
	if rc := sqlite3.Xsqlite3_vfs_register(tls, p, 0); rc != sqlite3.SQLITE_OK {
		return fmt.Errorf("cryptvfs: register %s: %d", name, rc)
	}
	return nil
}

func initMethods() {
	methods = sqlite3.Tsqlite3_io_methods{
		FiVersion:               2,
		FxClose:                 fp(xClose),
		FxRead:                  fp(xRead),
		FxWrite:                 fp(xWrite),
		FxTruncate:              fp(xTruncate),
		FxSync:                  fp(xSync),
		FxFileSize:              fp(xFileSize),
		FxLock:                  fp(xLock),
		FxUnlock:                fp(xUnlock),
		FxCheckReservedLock:     fp(xCheckReservedLock),
		FxFileControl:           fp(xFileControl),
		FxSectorSize:            fp(xSectorSize),
		FxDeviceCharacteristics: fp(xDeviceCharacteristics),
		FxShmMap:                fp(xShmMap),
		FxShmLock:               fp(xShmLock),
		FxShmBarrier:            fp(xShmBarrier),
		FxShmUnmap:              fp(xShmUnmap),
	}
}

func get(pFile uintptr) *state {
	h := *(*uintptr)(ptr(pFile + 8))
	mu.Lock()
	s := files[h]
	mu.Unlock()
	return s
}

func io(s *state) *sqlite3.Tsqlite3_io_methods {
	return (*sqlite3.Tsqlite3_io_methods)(ptr(s.methods))
}

func xOpen(tls *libc.TLS, pVfs uintptr, zName sqlite3.Tsqlite3_filename, pFile uintptr, flags int32, pOutFlags uintptr) int32 {
	app := (*[2]uintptr)(ptr((*sqlite3.Tsqlite3_vfs)(ptr(pVfs)).FpAppData))
	mu.Lock()
	e := vfses[app[1]]
	mu.Unlock()
	*(*uintptr)(ptr(pFile)) = 0
	*(*uintptr)(ptr(pFile + 8)) = 0
	sub := pFile + shimHdr
	baseOpen := (*sqlite3.Tsqlite3_vfs)(ptr(e.base)).FxOpen
	rc := (*(*func(*libc.TLS, uintptr, sqlite3.Tsqlite3_filename, uintptr, int32, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{baseOpen})))(tls, e.base, zName, sub, flags, pOutFlags)
	if rc != sqlite3.SQLITE_OK {
		return rc
	}
	s := &state{sub: sub, methods: (*sqlite3.Tsqlite3_file)(ptr(sub)).FpMethods}
	switch {
	case flags&sqlite3.SQLITE_OPEN_MAIN_DB != 0:
		s.kind, s.xts = kindDB, e.db
	case flags&sqlite3.SQLITE_OPEN_WAL != 0:
		s.kind, s.xts = kindWAL, e.wal
	case zName != 0 && flags&(sqlite3.SQLITE_OPEN_MAIN_JOURNAL|sqlite3.SQLITE_OPEN_SUPER_JOURNAL) != 0:
		s.kind, s.xts = kindJournal, e.jrn
	default:
		k := make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return sqlite3.SQLITE_IOERR
		}
		b, err := aes.NewCipher(k)
		if err != nil {
			return sqlite3.SQLITE_IOERR
		}
		s.kind, s.temp = kindTemp, b
	}
	s.scratch = libc.Xmalloc(tls, PageSize+frameHdr)
	if s.scratch == 0 {
		return sqlite3.SQLITE_NOMEM
	}
	mu.Lock()
	h := handle()
	files[h] = s
	mu.Unlock()
	*(*uintptr)(ptr(pFile + 8)) = h
	*(*uintptr)(ptr(pFile)) = uintptr(unsafe.Pointer(&methods))
	return sqlite3.SQLITE_OK
}

func mem(p uintptr, n int32) []byte {
	return unsafe.Slice((*byte)(ptr(p)), n)
}

func baseRead(tls *libc.TLS, s *state, buf uintptr, amt int32, off int64) int32 {
	return (*(*func(*libc.TLS, uintptr, uintptr, int32, sqlite3.Tsqlite3_int64) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxRead})))(tls, s.sub, buf, amt, off)
}

func baseWrite(tls *libc.TLS, s *state, buf uintptr, amt int32, off int64) int32 {
	return (*(*func(*libc.TLS, uintptr, uintptr, int32, sqlite3.Tsqlite3_int64) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxWrite})))(tls, s.sub, buf, amt, off)
}

func zero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

func ctr(s *state, b []byte, off int64) {
	iv := make([]byte, 16)
	binary.BigEndian.PutUint64(iv[8:], uint64(off/16))
	stream := cipher.NewCTR(s.temp, iv)
	skip := int(off % 16)
	if skip > 0 {
		pad := make([]byte, skip)
		stream.XORKeyStream(pad, pad)
	}
	stream.XORKeyStream(b, b)
}

func xRead(tls *libc.TLS, pFile uintptr, zBuf uintptr, iAmt int32, iOfst sqlite3.Tsqlite3_int64) int32 {
	s := get(pFile)
	off := int64(iOfst)
	switch s.kind {
	case kindTemp:
		rc := baseRead(tls, s, zBuf, iAmt, off)
		if rc == sqlite3.SQLITE_OK {
			ctr(s, mem(zBuf, iAmt), off)
		}
		return rc
	case kindDB:
		if iAmt == PageSize && off%PageSize == 0 {
			rc := baseRead(tls, s, zBuf, iAmt, off)
			if rc == sqlite3.SQLITE_OK {
				b := mem(zBuf, iAmt)
				s.xts.Decrypt(b, b, uint64(off))
			}
			return rc
		}
		out := mem(zBuf, iAmt)
		short := false
		for done := int64(0); done < int64(iAmt); {
			pos := off + done
			page := pos / PageSize * PageSize
			rc := baseRead(tls, s, s.scratch, PageSize, page)
			b := mem(s.scratch, PageSize)
			switch rc {
			case sqlite3.SQLITE_OK:
				s.xts.Decrypt(b, b, uint64(page))
			case sqlite3.SQLITE_IOERR_SHORT_READ:
				short = true
				if !zero(b) {
					return sqlite3.SQLITE_IOERR_READ
				}
			default:
				return rc
			}
			c := copy(out[done:], b[pos-page:])
			done += int64(c)
		}
		if short {
			return sqlite3.SQLITE_IOERR_SHORT_READ
		}
		return sqlite3.SQLITE_OK
	default:
		rc := baseRead(tls, s, zBuf, iAmt, off)
		if rc != sqlite3.SQLITE_OK {
			return rc
		}
		b := mem(zBuf, iAmt)
		switch {
		case iAmt == PageSize && encrypted(s, off):
			s.xts.Decrypt(b, b, uint64(off))
		case s.kind == kindWAL && iAmt == PageSize+frameHdr:
			p := b[frameHdr:]
			s.xts.Decrypt(p, p, uint64(off+frameHdr))
		}
		return rc
	}
}

func encrypted(s *state, off int64) bool {
	if s.kind == kindJournal {
		return off%8 == 4
	}
	return true
}

func xWrite(tls *libc.TLS, pFile uintptr, zBuf uintptr, iAmt int32, iOfst sqlite3.Tsqlite3_int64) int32 {
	s := get(pFile)
	off := int64(iOfst)
	switch s.kind {
	case kindTemp:
		var tmp uintptr
		if iAmt <= PageSize+frameHdr {
			tmp = s.scratch
		} else {
			tmp = libc.Xmalloc(tls, libc.Tsize_t(iAmt))
			if tmp == 0 {
				return sqlite3.SQLITE_NOMEM
			}
			defer libc.Xfree(tls, tmp)
		}
		b := mem(tmp, iAmt)
		copy(b, mem(zBuf, iAmt))
		ctr(s, b, off)
		return baseWrite(tls, s, tmp, iAmt, off)
	case kindDB:
		if iAmt == PageSize && off%PageSize == 0 {
			b := mem(s.scratch, PageSize)
			s.xts.Encrypt(b, mem(zBuf, iAmt), uint64(off))
			return baseWrite(tls, s, s.scratch, iAmt, off)
		}
		in := mem(zBuf, iAmt)
		for done := int64(0); done < int64(iAmt); {
			pos := off + done
			page := pos / PageSize * PageSize
			rc := baseRead(tls, s, s.scratch, PageSize, page)
			b := mem(s.scratch, PageSize)
			switch rc {
			case sqlite3.SQLITE_OK:
				s.xts.Decrypt(b, b, uint64(page))
			case sqlite3.SQLITE_IOERR_SHORT_READ:
				clear(b)
			default:
				return rc
			}
			c := copy(b[pos-page:], in[done:])
			s.xts.Encrypt(b, b, uint64(page))
			if rc := baseWrite(tls, s, s.scratch, PageSize, page); rc != sqlite3.SQLITE_OK {
				return rc
			}
			done += int64(c)
		}
		return sqlite3.SQLITE_OK
	default:
		if iAmt == PageSize && encrypted(s, off) {
			b := mem(s.scratch, PageSize)
			s.xts.Encrypt(b, mem(zBuf, iAmt), uint64(off))
			return baseWrite(tls, s, s.scratch, iAmt, off)
		}
		if s.kind == kindWAL && iAmt == PageSize+frameHdr {
			b := mem(s.scratch, iAmt)
			src := mem(zBuf, iAmt)
			for i := range frameHdr {
				b[i] = src[i]
			}
			s.xts.Encrypt(b[frameHdr:], src[frameHdr:], uint64(off+frameHdr))
			return baseWrite(tls, s, s.scratch, iAmt, off)
		}
		return baseWrite(tls, s, zBuf, iAmt, off)
	}
}

func xClose(tls *libc.TLS, pFile uintptr) int32 {
	s := get(pFile)
	rc := (*(*func(*libc.TLS, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxClose})))(tls, s.sub)
	mu.Lock()
	delete(files, *(*uintptr)(ptr(pFile + 8)))
	mu.Unlock()
	libc.Xfree(tls, s.scratch)
	*(*uintptr)(ptr(pFile)) = 0
	return rc
}

func xTruncate(tls *libc.TLS, pFile uintptr, size sqlite3.Tsqlite3_int64) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, sqlite3.Tsqlite3_int64) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxTruncate})))(tls, s.sub, size)
}

func xSync(tls *libc.TLS, pFile uintptr, flags int32) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, int32) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxSync})))(tls, s.sub, flags)
}

func xFileSize(tls *libc.TLS, pFile uintptr, pSize uintptr) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxFileSize})))(tls, s.sub, pSize)
}

func xLock(tls *libc.TLS, pFile uintptr, lock int32) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, int32) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxLock})))(tls, s.sub, lock)
}

func xUnlock(tls *libc.TLS, pFile uintptr, lock int32) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, int32) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxUnlock})))(tls, s.sub, lock)
}

func xCheckReservedLock(tls *libc.TLS, pFile uintptr, pResOut uintptr) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxCheckReservedLock})))(tls, s.sub, pResOut)
}

func xFileControl(tls *libc.TLS, pFile uintptr, op int32, pArg uintptr) int32 {
	s := get(pFile)
	if op == sqlite3.SQLITE_FCNTL_MMAP_SIZE {
		if pArg != 0 {
			*(*int64)(ptr(pArg)) = 0
		}
		return sqlite3.SQLITE_OK
	}
	return (*(*func(*libc.TLS, uintptr, int32, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxFileControl})))(tls, s.sub, op, pArg)
}

func xSectorSize(tls *libc.TLS, pFile uintptr) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxSectorSize})))(tls, s.sub)
}

func xDeviceCharacteristics(tls *libc.TLS, pFile uintptr) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxDeviceCharacteristics})))(tls, s.sub)
}

func xShmMap(tls *libc.TLS, pFile uintptr, iPg, pgsz, bExtend int32, pp uintptr) int32 {
	s := get(pFile)
	if io(s).FiVersion < 2 || io(s).FxShmMap == 0 {
		return sqlite3.SQLITE_IOERR
	}
	return (*(*func(*libc.TLS, uintptr, int32, int32, int32, uintptr) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxShmMap})))(tls, s.sub, iPg, pgsz, bExtend, pp)
}

func xShmLock(tls *libc.TLS, pFile uintptr, offset, n, flags int32) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, int32, int32, int32) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxShmLock})))(tls, s.sub, offset, n, flags)
}

func xShmBarrier(tls *libc.TLS, pFile uintptr) {
	s := get(pFile)
	(*(*func(*libc.TLS, uintptr))(unsafe.Pointer(&struct{ uintptr }{io(s).FxShmBarrier})))(tls, s.sub)
}

func xShmUnmap(tls *libc.TLS, pFile uintptr, deleteFlag int32) int32 {
	s := get(pFile)
	return (*(*func(*libc.TLS, uintptr, int32) int32)(unsafe.Pointer(&struct{ uintptr }{io(s).FxShmUnmap})))(tls, s.sub, deleteFlag)
}
