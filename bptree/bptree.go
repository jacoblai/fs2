package bptree

import (
	"reflect"
)

import (
	"github.com/timtadh/fs2/errors"
	"github.com/timtadh/fs2/fmap"
	"github.com/timtadh/fs2/slice"
)

type BpTree struct {
	bf *fmap.BlockFile
	metaBack []byte
	meta *bpTreeMeta
	cacheCert uintptr
	leafCache map[uint64]*leaf
	internalCache map[uint64]*internal
}

type bpTreeMeta struct {
	root uint64
	keySize uint16
}

var bpTreeMetaSize uintptr

func init() {
	m := &bpTreeMeta{}
	bpTreeMetaSize = reflect.TypeOf(*m).Size()
}

func newBpTreeMeta(bf *fmap.BlockFile, keySize uint16) ([]byte, *bpTreeMeta, error) {
	a, err := bf.Allocate()
	if err != nil {
		return nil, nil, err
	}
	err = bf.Do(a, 1, func(bytes []byte) error {
		_, err := newLeaf(bytes, keySize)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	data := make([]byte, bpTreeMetaSize)
	meta := (*bpTreeMeta)(slice.AsSlice(&data).Array)
	meta.root = a
	meta.keySize = keySize
	err = bf.SetControlData(data)
	if err != nil {
		return nil, nil, err
	}
	return data, meta, nil
}

func loadBpTreeMeta(bf *fmap.BlockFile) ([]byte, *bpTreeMeta, error) {
	data, err := bf.ControlData()
	if err != nil {
		return nil, nil, err
	}
	meta := (*bpTreeMeta)(slice.AsSlice(&data).Array)
	if meta.root == 0 || meta.keySize == 0 {
		return nil, nil, errors.Errorf("Meta was not properly initialized. Can't load tree")
	}
	return data, meta, nil
}

func New(bf *fmap.BlockFile, keySize int) (*BpTree, error) {
	if keysPerInternal(int(bf.BlockSize()), keySize) < 3 {
		return nil, errors.Errorf("Key is too large (fewer than 3 keys per internal node)")
	}
	back, meta, err := newBpTreeMeta(bf, uint16(keySize))
	if err != nil {
		return nil, err
	}
	bpt := &BpTree{
		bf: bf,
		metaBack: back,
		meta: meta,
		leafCache: make(map[uint64]*leaf),
		internalCache: make(map[uint64]*internal),
	}
	return bpt, nil
}

func Open(bf *fmap.BlockFile) (*BpTree, error) {
	back, meta, err := loadBpTreeMeta(bf)
	if err != nil {
		return nil, err
	}
	bpt := &BpTree{
		bf: bf,
		metaBack: back,
		meta: meta,
		leafCache: make(map[uint64]*leaf),
		internalCache: make(map[uint64]*internal),
	}
	return bpt, nil
}

func (b *BpTree) writeMeta() error {
	return b.bf.SetControlDataNoSync(b.metaBack)
}

func (self *BpTree) KeySize() int {
	return int(self.meta.keySize)
}

