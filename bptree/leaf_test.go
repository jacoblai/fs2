package bptree

import "testing"

import (
	"bytes"
	// "fmt"
	"os"
	"runtime/debug"
	"unsafe"
)

import (
	"github.com/timtadh/fs2/slice"
)

const TESTS = 50

type T testing.T

func (t *T) Log(msgs ...interface{}) {
	x := (*testing.T)(t)
	// fmt.Println(msgs...)
	x.Log(msgs...)
}

func (t *T) assert(msg string, oks ...bool) {
	for _, ok := range oks {
		if !ok {
			t.Log("\n"+string(debug.Stack()))
			t.Error(msg)
			t.Fatal("assert failed")
		}
	}
}

func (t *T) assert_nil(errors ...error) {
	for _, err := range errors {
		if err != nil {
			t.Log("\n"+string(debug.Stack()))
			t.Fatal(err)
		}
	}
}

func (t *T) rand_bytes(length int) []byte {
	if urandom, err := os.Open("/dev/urandom"); err != nil {
		t.Fatal(err)
	} else {
		slice := make([]byte, length)
		if _, err := urandom.Read(slice); err != nil {
			t.Fatal(err)
		}
		urandom.Close()
		return slice
	}
	panic("unreachable")
}

func (t *T) rand_key() []byte {
	return t.rand_bytes(8)
}

func (t *T) rand_value(max int) []byte {
	bytes := t.rand_bytes(2)
	s := slice.AsSlice(&bytes)
	length := int(*(*uint16)(s.Array))
	length = (length % (max))
	return t.rand_bytes(length)
}

func (t *T) rand_bigValue(min, max int) []byte {
	bytes := t.rand_bytes(4)
	s := slice.AsSlice(&bytes)
	length := int(*(*uint32)(s.Array))
	length = (length % (max)) + min
	return t.rand_bytes(length)
}

func (t *T) bkey(key *uint64) []byte {
	s := &slice.Slice{
		Array: unsafe.Pointer(key),
		Len: 8,
		Cap: 8,
	}
	return *s.AsBytes()
}

func (t *T) key(bytes []byte) uint64 {
	t.assert("bytes must have length 8", len(bytes) == 8)
	s := slice.AsSlice(&bytes)
	return *(*uint64)(s.Array)
}

func (t *T) newLeaf() *leaf {
	n, err := newLeaf(testAlloc(), 8)
	t.assert_nil(err)
	return n
}

func TestPutKVRand(x *testing.T) {
	t := (*T)(x)
	bf, bf_clean := t.blkfile()
	for TEST := 0; TEST < TESTS; TEST++ {
		n, err := newLeaf(make([]byte, 1027+TEST*16), 8)
		t.assert_nil(err)
		do_put := func(kv *KV) {
			t.assert_nil(n.putKV(SMALL_VALUE, kv.key, kv.value))
		}
		kvs := make([]*KV, 0, n.meta.keyCap/2)
		// t.Log(n)
		for i := 0; i < cap(kvs); i++ {
			kv := t.make_kv()
			if !n.fits(kv.value) {
				break
			}
			kvs = append(kvs, kv)
			do_put(kv)
			// t.Log(n)
			t.assert("could not find key in leaf", n.Has(kv.key))
			t.assert_value(kv.value)(n.first_value(bf, kv.key))
		}
		for _, kv := range kvs {
			t.assert("could not find key in leaf", n.Has(kv.key))
			t.assert_value(kv.value)(n.first_value(bf, kv.key))
		}
	}
	bf_clean()
}

func TestPutDelKVRand(x *testing.T) {
	t := (*T)(x)
	bf, bf_clean := t.blkfile()
	for TEST := 0; TEST < TESTS*2; TEST++ {
		n, err := newLeaf(make([]byte, 1027+TEST*16), 8)
		t.assert_nil(err)
		kvs := make([]*KV, 0, n.meta.keyCap/2)
		// t.Log(n)
		for i := 0; i < cap(kvs); i++ {
			kv := t.make_kv()
			if !n.fits(kv.value) {
				break
			}
			kvs = append(kvs, kv)
			t.assert_nil(n.putKV(SMALL_VALUE, kv.key, kv.value))
			t.assert("could not find key in leaf", n.Has(kv.key))
			t.assert_value(kv.value)(n.first_value(bf, kv.key))
		}
		for _, kv := range kvs {
			t.assert("could not find key in leaf", n.Has(kv.key))
		}
		for i, kv := range kvs {
			t.assert_nil(n.delKV(kv.key, func(b []byte) bool {
				return bytes.Equal(b, kv.value)
			}))
			for _, kv2 := range kvs[:i+1] {
				t.assert("found key in leaf", !n.Has(kv2.key))
			}
		}
		for _, kv := range kvs {
			t.assert_nil(n.putKV(SMALL_VALUE, kv.key, kv.value))
			t.assert("could not find key in leaf", n.Has(kv.key))
			t.assert_value(kv.value)(n.first_value(bf, kv.key))
		}
		for _, kv := range kvs {
			t.assert("could not find key in leaf", n.Has(kv.key))
			t.assert_value(kv.value)(n.first_value(bf, kv.key))
		}
		for _, kv := range kvs {
			t.assert_nil(n.delKV(kv.key, func(b []byte) bool {
				return bytes.Equal(b, kv.value)
			}))
			for _, kv2 := range kvs {
				if !bytes.Equal(kv.key, kv2.key) {
					t.assert("no key in leaf", n.Has(kv2.key))
				}
			}
			t.assert("found key in leaf", !n.Has(kv.key))
			t.assert_nil(n.putKV(SMALL_VALUE, kv.key, kv.value))
		}
		for i, kv := range kvs {
			t.assert_nil(n.delKV(kv.key, func(b []byte) bool {
				return bytes.Equal(b, kv.value)
			}))
			for _, kv2 := range kvs[:i+1] {
				t.assert("found key in leaf", !n.Has(kv2.key))
			}
		}
	}
	bf_clean()
}

func TestPutKV(x *testing.T) {
	t := (*T)(x)
	bf, bf_clean := t.blkfile()
	n := t.newLeaf()
	k1 := uint64(7)
	v1 := []byte{7,7,7,7,7,7,7,7}
	k2 := uint64(3)
	v2 := []byte{3,3,3}
	k3 := uint64(12)
	v3 := []byte{12,12,12,12,12,12,12,12}
	k4 := uint64(8)
	v4 := []byte{8,8}
	k5 := uint64(5)
	v5 := []byte{5,5,5,5,5,5,5,5,5,5,5}
	t.assert_nil(n.putKV(SMALL_VALUE, t.bkey(&k1), v1))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k1)))
	t.assert_value(v1)(n.first_value(bf, t.bkey(&k1)))

	t.assert_nil(n.putKV(SMALL_VALUE, t.bkey(&k2), v2))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k2)))
	t.assert_value(v2)(n.first_value(bf, t.bkey(&k2)))

	t.assert_nil(n.putKV(SMALL_VALUE, t.bkey(&k3), v3))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k3)))
	t.assert_value(v3)(n.first_value(bf, t.bkey(&k3)))

	t.assert_nil(n.putKV(SMALL_VALUE, t.bkey(&k4), v4))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k4)))
	t.assert_value(v4)(n.first_value(bf, t.bkey(&k4)))

	t.assert_nil(n.putKV(SMALL_VALUE, t.bkey(&k5), v5))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k5)))
	t.assert_value(v5)(n.first_value(bf, t.bkey(&k5)))

	t.assert("could not find key in leaf", n.Has(t.bkey(&k1)))
	t.assert_value(v1)(n.first_value(bf, t.bkey(&k1)))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k2)))
	t.assert_value(v2)(n.first_value(bf, t.bkey(&k2)))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k3)))
	t.assert_value(v3)(n.first_value(bf, t.bkey(&k3)))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k4)))
	t.assert_value(v4)(n.first_value(bf, t.bkey(&k4)))
	t.assert("could not find key in leaf", n.Has(t.bkey(&k5)))
	t.assert_value(v5)(n.first_value(bf, t.bkey(&k5)))
	bf_clean()
}

func TestNewLeaf(t *testing.T) {
	n, err := newLeaf(testAlloc(), 16)
	if err != nil {
		t.Fatal(err)
	}
	if n.meta.flags != LEAF {
		t.Error("was not a LEAF node")
	}
	if n.meta.keySize != 16 {
		t.Error("keySize was not 16")
	}
	if n.meta.keyCap != 5 {
		t.Error("keyCap was not 5")
	}
	if n.meta.keyCount != 0 {
		t.Error("keyCount was not 0")
	}
	for i := 0; i < int(n.meta.keyCap); i++ {
		if n.valueSizes[i] != 0 {
			t.Error("ptr was not zero")
		}
	}
	
	n.valueSizes[0] = 1
	n.valueSizes[1] = 21
	n.valueSizes[2] = 23
	n.valueSizes[3] = 125
	n.valueSizes[n.meta.keyCap-1] = 0xffff

	valueSizes := []uint16{1, 21, 23, 125, 0xffff}

	if n.meta.flags != LEAF {
		t.Error("was not an leaf node")
	}
	if n.meta.keySize != 16 {
		t.Error("keySize was not 16")
	}
	if n.meta.keyCap != 5 {
		t.Error("keyCap was not 5")
	}
	if n.meta.keyCount != 0 {
		t.Error("keyCount was not 0")
	}

	for i := 0; i < int(n.meta.keyCap); i++ {
		if n.valueSizes[i] != valueSizes[i] {
			t.Error("valueSize was not the correct value")
		}
	}
}

func TestLoadLeaf(t *testing.T) {
	back := func() []byte {
		n, err := newLeaf(testAlloc(), 16)
		if err != nil {
			t.Fatal(err)
		}
		n.valueSizes[0] = 1
		n.valueSizes[1] = 21
		n.valueSizes[2] = 23
		n.valueSizes[3] = 125
		n.valueSizes[n.meta.keyCap-1] = 0xffff
		return n.back
	}()

	n, err := loadLeaf(back)
	if err != nil {
		t.Fatal(err)
	}

	valueSizes := []uint16{1, 21, 23, 125, 0xffff}

	if n.meta.flags != LEAF {
		t.Error("was not an leaf node")
	}
	if n.meta.keySize != 16 {
		t.Error("keySize was not 16")
	}
	if n.meta.keyCap != 5 {
		t.Error("keyCap was not 5")
	}
	if n.meta.keyCount != 0 {
		t.Error("keyCount was not 0")
	}

	for i := 0; i < int(n.meta.keyCap); i++ {
		if n.valueSizes[i] != valueSizes[i] {
			t.Error("ptr was not the correct value")
		}
	}
}
