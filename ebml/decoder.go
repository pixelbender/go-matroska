package matroska

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"reflect"
	"time"
	"log"
)

var ErrFormat = errors.New("ebml: not a valid format")
var ErrReadSequence = errors.New("ebml: read error")

// See https://matroska.org/files/matroska.pdf

var mask = []byte{0x80, 0x40, 0x20, 0x10, 0x8, 0x4, 0x2, 0x1}
var rest = []byte{0xff, 0x7f, 0x3f, 0x1f, 0xf, 0x7, 0x3, 0x1, 0x0}
var maxBufferSize = int64(1 << 20)
var absTime = time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)

type Decoder struct {
	rs   io.ReadSeeker
	buf  *bufio.Reader
	len  int64
	size int64
	id   int64
	elem *Decoder
}

func NewDecoder(r io.Reader) *Decoder {
	dec := new(Decoder)
	if rs, ok := r.(io.ReadSeeker); ok {
		dec.rs = rs
	}
	if buf, ok := r.(*bufio.Reader); ok {
		dec.buf = buf
	} else {
		dec.buf = bufio.NewReader(r)
	}
	if f, ok := r.(*os.File); ok {
		if info, err := f.Stat(); err == nil {
			dec.size = info.Size()
		}
	}
	dec.len = dec.size
	return dec
}

func (dec *Decoder) Size() int64 {
	dec.skip()
	return dec.len
}

func (dec *Decoder) Next() (id int64, v *Decoder, err error) {
	if err = dec.skip(); err != nil {
		return
	}
	var size int64
	if id, _, err = dec.readVint(0); err != nil {
		return
	}
	if size, _, err = dec.readVint(1); err != nil {
		return
	}
	if id == 0 || size < 0 {
		err = ErrFormat
		return
	}
	if dec.size < size && 0 < dec.size {
		err = ErrFormat
		return
	}
	if size == 0 {
		log.Printf("Warning empty element: 0x%x", id)
		return
	}
	dec.len -= size
	v = &Decoder{dec.rs, dec.buf, size, size, id, nil}
	dec.elem = v
	return
}

// TODO: clarify errors
func (dec *Decoder) Decode(v interface{}) (err error) {
	if err = dec.skip(); err != nil {
		return
	}
	if u, ok := v.(Unmarshaler); ok {
		return u.UnmarshalEBML(dec)
	}
	if v == nil {
		return errors.New("ebml: Decode nil")
	}
	ref := reflect.ValueOf(v)
	if ref.Kind() != reflect.Ptr {
		return errors.New("ebml: Decode not a pointer")
	}
	if ref = ref.Elem(); ref.Kind() != reflect.Struct {
		return errors.New("ebml: Decode not a struct")
	}
	u := &typeCodec{ref}
	return u.UnmarshalEBML(dec)
}

func (dec *Decoder) Read(b []byte) (n int, err error) {
	if err = dec.skip(); err != nil {
		return
	}
	if m := int64(len(b)); dec.len < m && 0 < dec.size {
		b = b[:dec.len]
	}
	for len(b) > 0 {
		var c int
		if c, err = dec.buf.Read(b); err != nil {
			return
		}
		if c > 0 {
			n += c
			dec.len -= int64(c)
			b = b[c:]
		} else {
			err = io.EOF
			return
		}
	}
	return
}

func (dec *Decoder) skip() (err error) {
	if e := dec.elem; e != nil {
		err = dec.elem.Skip()
		if err != nil {
			return err
		}
	}
	return
}

func (dec *Decoder) Skip() (err error) {
	if err = dec.skip(); err != nil {
		return
	}
	if dec.len <= 0 {
		return
	}
	n := int64(dec.buf.Buffered())
	log.Printf("Skipping: 0x%x %v bytes", dec.id, dec.len)
	if dec.rs != nil && dec.len > n {
		if _, err = dec.rs.Seek(dec.len - n, 1); err != nil {
			return
		}
		dec.buf.Reset(dec.rs)
	} else {
		if _, err = dec.buf.Discard(int(dec.len)); err != nil {
			return
		}
	}
	dec.len = 0
	return
}

func (dec *Decoder) ReadInt() (v int64, err error) {
	if err = dec.skip(); err != nil {
		return
	}
	if dec.len < 1 || dec.len > 8 {
		err = ErrFormat
		return
	}
	n := int(dec.len)
	var b []byte
	if b, err = dec.buf.Peek(n); err != nil {
		return
	}
	for _, it := range b {
		v = (v << 8) | int64(it)
	}
	if _, err = dec.buf.Discard(n); err != nil {
		return
	}
	dec.len = 0
	return
}

func (dec *Decoder) ReadFloat() (v float64, err error) {
	if err = dec.skip(); err != nil {
		return
	}
	var b []byte
	switch dec.len {
	case 4:
		if b, err = dec.buf.Peek(4); err != nil {
			return
		}
		v = float64(math.Float32frombits(binary.BigEndian.Uint32(b)))
	case 8:
		if b, err = dec.buf.Peek(8); err != nil {
			return
		}
		v = math.Float64frombits(binary.BigEndian.Uint64(b))
	default:
		err = ErrFormat
		return
	}
	if _, err = dec.buf.Discard(int(dec.len)); err != nil {
		return
	}
	dec.len = 0
	return
}

func (dec *Decoder) ReadBool() (b bool, err error) {
	var v int64
	if v, err = dec.ReadInt(); err != nil {
		return
	}
	b = v != 0
	return
}

func (dec *Decoder) ReadTime() (t time.Time, err error) {
	var v int64
	if v, err = dec.ReadInt(); err != nil {
		return
	}
	t = absTime.Add(time.Duration(v) * time.Nanosecond)
	return
}

func (dec *Decoder) ReadString() (v string, err error) {
	var b []byte
	b, err = dec.ReadBytes()
	if err != nil {
		return
	}
	v = string(b)
	return
}

func (dec *Decoder) ReadBytes() (v []byte, err error) {
	if err = dec.skip(); err != nil {
		return
	}
	if dec.len < 0 || maxBufferSize < dec.len {
		err = ErrFormat
		return
	}
	b := make([]byte, dec.len)
	if _, err = dec.Read(b); err != nil {
		return
	}
	v = b
	return
}

func (dec *Decoder) readVint(off int) (v int64, n int, err error) {
	if err = dec.skip(); err != nil {
		return
	}
	if dec.len < 1 && 0 < dec.size {
		err = io.EOF
		return
	}
	m, err := dec.buf.ReadByte()
	if err != nil {
		return
	}
	dec.len--
	var bit byte
	for n, bit = range mask {
		if m & bit != 0 {
			v = int64(m & rest[n + off])
			break
		}
	}
	if n > 0 {
		if dec.len < int64(n) && 0 < dec.size {
			err = io.EOF
			return
		}
		var b []byte
		if b, err = dec.buf.Peek(n); err != nil {
			return
		}
		for _, it := range b {
			v = (v << 8) | int64(it)
		}
		if _, err = dec.buf.Discard(n); err != nil {
			return
		}
		dec.len -= int64(n)
	}
	return
}
