package ebml

import (
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO: use cache
var typeMap = make(map[reflect.Type]*structInfo)
var typeLock sync.RWMutex

var timeType = reflect.TypeOf(time.Time{})

type typeCodec struct {
	v reflect.Value
}

var prefix int

func (c *typeCodec) UnmarshalEBML(dec *Decoder) error {
	prefix++
	defer func() {
		prefix--
	}()

	s, err := newStructInfo(c.v.Type())
	if err != nil {
		return err
	}
	for {
		id, elem, err := dec.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if elem == nil {
			continue
		}

		if f, ok := s.ids[id]; ok {
			if err = f.Decode(c.v.Field(f.index), elem); err != nil {
				return err
			}
		} else if err = elem.Skip(); err != nil {
			return err
		}
	}
	return nil
}

type structInfo struct {
	fields []*fieldInfo
	ids    map[int64]*fieldInfo
}

func newStructInfo(t reflect.Type) (c *structInfo, err error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New("ebml: Decode not a struct")
	}
	n := t.NumField()
	c = &structInfo{
		fields: make([]*fieldInfo, 0, n),
		ids : make(map[int64]*fieldInfo),
	}
	var id int64
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}
		tag := f.Tag.Get("ebml")
		if tag == "" || tag == "-" {
			continue
		}
		if f.Anonymous {
			// TODO: implement
			continue
		}
		p := strings.Split(tag, ",")
		seq := strings.Split(p[0], ">")

		var it *fieldInfo

		for _, s := range seq {
			if id, err = strconv.ParseInt(s, 16, 64); err != nil {
				// TODO: clearify error message
				return
			}
			if it == nil {
				it = &fieldInfo{id, i, f.Name, nil}
			} else {
				it.seq = append(it.seq, id)
			}
		}

		c.fields = append(c.fields, it)
		c.ids[it.id] = it
	}
	return
}

type fieldInfo struct {
	id    int64
	index int
	name  string
	seq   []int64
}

func (f *fieldInfo) decodeSeq(seq []int64, v reflect.Value, dec *Decoder) error {
	for {
		id, elem, err := dec.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if elem == nil {
			continue
		}
		if id == seq[0] {
			if len(seq) == 1 {
				err = f.decode(v, elem)
			} else {
				err = f.decodeSeq(seq[1:], v, elem)
			}
			if err != nil {
				return err
			}
		} else if err = elem.Skip(); err != nil {
			return err
		}
	}
	return nil
}

func (f *fieldInfo) Decode(v reflect.Value, dec *Decoder) error {
	if f.seq != nil {
		return f.decodeSeq(f.seq, v, dec)
	}
	return f.decode(v, dec)
}

func (f *fieldInfo) decode(v reflect.Value, dec *Decoder) error {
	switch v.Kind() {
	case reflect.Struct:
		if v.Type() == timeType {
			if t, err := dec.ReadTime(); err != nil {
				return err
			} else {
				v.Set(reflect.ValueOf(t))
			}
			return nil
		}
		u := &typeCodec{v}
		return u.UnmarshalEBML(dec)

	case reflect.Ptr:
		e := v.Type().Elem()
		if e.Kind() != reflect.Struct {
			return errors.New("ebml: unsupported pointer type " + e.String())
		}
		if v.IsNil() {
			v.Set(reflect.New(e))
		}
		if u, ok := v.Interface().(Unmarshaler); ok {
			return dec.Decode(u)
		}
		return f.decode(v.Elem(), dec)

	case reflect.Slice:
		e := v.Type().Elem()

		if e.Kind() == reflect.Uint8 {
			if b, err := dec.ReadBytes(); err != nil {
				return err
			} else {
				v.SetBytes(b)
			}
			return nil
		} else if e.Kind() == reflect.Int64 {
			if i, err := dec.ReadInt(); err != nil {
				return err
			} else {
				v.Set(reflect.Append(v, reflect.ValueOf(i)))
			}
			return nil
		}

		if e.Kind() != reflect.Ptr {
			return errors.New("ebml: unsupported slice type " + e.String())
		}
		it := reflect.New(e.Elem())
		v.Set(reflect.Append(v, it))

		if u, ok := it.Interface().(Unmarshaler); ok {
			return dec.Decode(u)
		}

		return f.decode(it.Elem(), dec)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := dec.ReadInt(); err != nil {
			return err
		} else {
			v.SetInt(i)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if i, err := dec.ReadInt(); err != nil {
			return err
		} else {
			v.SetUint(uint64(i))
		}
	case reflect.Bool:
		if b, err := dec.ReadBool(); err != nil {
			return err
		} else {
			v.SetBool(b)
		}
	case reflect.Float32, reflect.Float64:
		if f, err := dec.ReadFloat(); err != nil {
			return err
		} else {
			v.SetFloat(f)
		}
	case reflect.String:
		if s, err := dec.ReadString(); err != nil {
			return err
		} else {
			v.SetString(s)
		}
	default:
		return errors.New("ebml: unsupported type " + v.Kind().String())
	}
	return nil
}
