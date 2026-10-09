package transfer

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/crypt"
)

const (
	maxEntries    = 50000
	maxDeclared   = 4 << 30
	maxEntryBytes = 256 << 20
	maxRatio      = 200
)

var errTooBig = errors.New("the archive is larger than imports allow")

type vfile struct {
	path string
	size int64
	open func() (io.ReadCloser, error)
}

type vfs struct {
	files   []*vfile
	byPath  map[string]*vfile
	byLower map[string]*vfile
	closers []io.Closer
	temps   []string
	dir     string
	key     *crypt.Key
}

func newVFS(dir string, key *crypt.Key) *vfs {
	return &vfs{byPath: map[string]*vfile{}, byLower: map[string]*vfile{}, dir: dir, key: key}
}

func (v *vfs) close() {
	for _, c := range v.closers {
		c.Close()
	}
	for _, t := range v.temps {
		os.Remove(t)
	}
}

func cleanPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = path.Clean("/" + p)
	return strings.TrimPrefix(p, "/")
}

func junk(p string) bool {
	base := path.Base(p)
	return strings.HasPrefix(p, "__MACOSX/") || strings.Contains(p, "/__MACOSX/") || base == ".DS_Store" || base == "Thumbs.db" || strings.HasPrefix(base, "._")
}

func (v *vfs) add(f *vfile) error {
	if len(v.files) >= maxEntries {
		return errTooBig
	}
	if _, dup := v.byPath[f.path]; dup {
		return nil
	}
	v.files = append(v.files, f)
	v.byPath[f.path] = f
	v.byLower[strings.ToLower(f.path)] = f
	return nil
}

func (v *vfs) addFile(name, diskPath string, size int64) error {
	if strings.EqualFold(path.Ext(name), ".zip") {
		return v.addZip(diskPath, "", 0)
	}
	return v.add(&vfile{path: cleanPath(path.Base(name)), size: size, open: func() (io.ReadCloser, error) { return v.key.Open(diskPath) }})
}

func (v *vfs) addZip(diskPath, prefix string, depth int) error {
	fh, err := v.key.Open(diskPath)
	if err != nil {
		return errors.New("the zip file could not be read")
	}
	v.closers = append(v.closers, fh)
	zr, err := zip.NewReader(fh, fh.Size())
	if err != nil {
		return errors.New("the zip file could not be read")
	}
	var declared uint64
	for _, e := range zr.File {
		if e.FileInfo().IsDir() {
			continue
		}
		p := cleanPath(path.Join(prefix, e.Name))
		if p == "" || junk(p) {
			continue
		}
		declared += e.UncompressedSize64
		if declared > maxDeclared {
			return errTooBig
		}
		if e.UncompressedSize64 > maxEntryBytes {
			continue
		}
		if e.CompressedSize64 > 0 && e.UncompressedSize64 > 50<<20 && e.UncompressedSize64/e.CompressedSize64 > maxRatio {
			continue
		}
		entry := e
		if strings.EqualFold(path.Ext(p), ".zip") && depth == 0 {
			tmp, err := os.CreateTemp(v.dir, ".import-*.zip")
			if err != nil {
				return err
			}
			v.temps = append(v.temps, tmp.Name())
			rc, err := entry.Open()
			if err != nil {
				tmp.Close()
				continue
			}
			enc := v.key.Writer(tmp)
			_, err = io.Copy(enc, io.LimitReader(rc, maxEntryBytes))
			if cerr := enc.Close(); err == nil {
				err = cerr
			}
			rc.Close()
			tmp.Close()
			if err != nil {
				continue
			}
			if err := v.addZip(tmp.Name(), path.Dir(p), depth+1); err != nil {
				if errors.Is(err, errTooBig) {
					return err
				}
			}
			continue
		}
		if err := v.add(&vfile{path: p, size: int64(entry.UncompressedSize64), open: func() (io.ReadCloser, error) {
			rc, err := entry.Open()
			if err != nil {
				return nil, err
			}
			return struct {
				io.Reader
				io.Closer
			}{io.LimitReader(rc, maxEntryBytes), rc}, nil
		}}); err != nil {
			return err
		}
	}
	return nil
}

func (v *vfs) read(f *vfile) ([]byte, error) {
	rc, err := f.open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, maxEntryBytes))
}

func (v *vfs) lookup(p string) *vfile {
	p = cleanPath(p)
	if f, ok := v.byPath[p]; ok {
		return f
	}
	return v.byLower[strings.ToLower(p)]
}

func (v *vfs) sorted() []*vfile {
	out := append([]*vfile(nil), v.files...)
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

func extOf(p string) string {
	return strings.ToLower(path.Ext(p))
}

func stripExt(name string) string {
	return strings.TrimSuffix(name, path.Ext(name))
}
