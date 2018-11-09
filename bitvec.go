package zcbit

import (
	"errors"
	"math/bits"
	"reflect"
	"unsafe"
)

const (
	wordBytes           = 8
	wordSize            = 64
	log2WordSize        = 6
	allBits      uint64 = 0xffffffffffffffff
)

// errors
var (
	ErrInvalidEndianness = errors.New("unsupported endianness")
	ErrInvalidLength     = errors.New("len(buffer) for zcbit must be N * 8")
	ErrUnsupportedArch   = errors.New("unsupported host endianness")
)

func swapUint64(n uint64) uint64 {
	return ((n & 0x00000000000000FF) << 56) |
		((n & 0x000000000000FF00) << 40) |
		((n & 0x0000000000FF0000) << 24) |
		((n & 0x00000000FF000000) << 8) |
		((n & 0x000000FF00000000) >> 8) |
		((n & 0x0000FF0000000000) >> 24) |
		((n & 0x00FF000000000000) >> 40) |
		((n & 0xFF00000000000000) >> 56)
}

// BitVec is bit vector component
type BitVec struct {
	vec  []uint64
	swap bool
}

// New create *BitVec
func New(b []byte, endian Endianness) (*BitVec, error) {
	if len(b)%8 != 0 {
		return nil, ErrInvalidLength
	} else if endian != LittleEndian && endian != BigEndian {
		return nil, ErrInvalidEndianness
	} else if hostEndian != LittleEndian && hostEndian != BigEndian {
		return nil, ErrUnsupportedArch
	}
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&b))
	header.Len /= wordBytes
	header.Cap /= wordBytes

	return &BitVec{
		vec:  *(*[]uint64)(unsafe.Pointer(&header)),
		swap: endian != hostEndian,
	}, nil
}

// Test checks the bit is set.
func (b *BitVec) Test(i uint) bool {
	idx := i >> log2WordSize
	if int(idx) >= len(b.vec) {
		return false
	}
	if b.swap {
		v := swapUint64(b.vec[idx])
		return v&(1<<(i&(wordSize-1))) != 0
	} else {
		return b.vec[idx]&(1<<(i&(wordSize-1))) != 0
	}
}

// Set sets 1 to bit
func (b *BitVec) Set(i uint) bool {
	idx := i >> log2WordSize
	if int(idx) >= len(b.vec) {
		return false
	}
	if b.swap {
		v := swapUint64(b.vec[idx])
		v |= 1 << (i & (wordSize - 1))
		b.vec[idx] = swapUint64(v)
	} else {
		b.vec[idx] |= 1 << (i & (wordSize - 1))
	}
	return true
}

// Clear sets 0 to bit
func (b *BitVec) Clear(i uint) bool {
	idx := i >> log2WordSize
	if int(idx) >= len(b.vec) {
		return false
	}
	if b.swap {
		v := swapUint64(b.vec[idx])
		v &^= 1 << (i & (wordSize - 1))
		b.vec[idx] = swapUint64(v)
	} else {
		b.vec[idx] &^= 1 << (i & (wordSize - 1))
	}
	return true
}

// FindFirstOne returns first 1 bit index and true.
// if not found then returns false
func (b *BitVec) FindFirstOne(i uint) (uint, bool) {
	idx := int(i >> log2WordSize)
	if idx >= len(b.vec) {
		return 0, false
	}
	if b.swap {
		v := swapUint64(b.vec[idx])
		v = v >> (i & (wordSize - 1))
		if v != 0 {
			return i + uint(bits.TrailingZeros64(v)), true
		}
		for idx++; idx < len(b.vec); idx++ {
			if b.vec[idx] != 0 {
				return uint(idx)*wordSize + uint(bits.TrailingZeros64(swapUint64(b.vec[idx]))), true
			}
		}
	} else {
		v := b.vec[idx] >> (i & (wordSize - 1))
		if v != 0 {
			return i + uint(bits.TrailingZeros64(v)), true
		}
		for idx++; idx < len(b.vec); idx++ {
			if b.vec[idx] != 0 {
				return uint(idx)*wordSize + uint(bits.TrailingZeros64(b.vec[idx])), true
			}
		}
	}
	return 0, false
}

// FindFirstZero returns first 0 bit index and true.
// if not found then returns false
// TODO: set tail
func (b *BitVec) FindFirstZero(i uint) (uint, bool) {
	idx := int(i >> log2WordSize)
	if idx >= len(b.vec) {
		return 0, false
	}
	if b.swap {
		offset := (i & (wordSize - 1))
		v := swapUint64(b.vec[idx]) >> offset
		trail := uint(bits.TrailingZeros64(^v))
		if trail < wordSize-offset {
			return i + trail, true
		}
		for idx++; idx < len(b.vec); idx++ {
			if b.vec[idx] != allBits {
				return uint(idx)*wordSize + uint(bits.TrailingZeros64(^swapUint64(b.vec[idx]))), true
			}
		}
	} else {
		offset := (i & (wordSize - 1))
		v := b.vec[idx] >> offset
		trail := uint(bits.TrailingZeros64(^v))
		if trail < wordSize-offset {
			return i + trail, true
		}
		for idx++; idx < len(b.vec); idx++ {
			if b.vec[idx] != allBits {
				return uint(idx)*wordSize + uint(bits.TrailingZeros64(^b.vec[idx])), true
			}
		}
	}
	return 0, false
}

// FindLastOne returns last 1 bit index and true.
// if not found then returns false
func (b *BitVec) FindLastOne() (uint, bool) {
	if b.swap {
		for i := len(b.vec); i > 0; i-- {
			v := b.vec[i-1]
			if v != 0 {
				v = swapUint64(v)
				return uint(i*wordSize - bits.LeadingZeros64(v) - 1), true
			}
		}
	} else {
		for i := len(b.vec); i > 0; i-- {
			v := b.vec[i-1]
			if v != 0 {
				return uint(i*wordSize - bits.LeadingZeros64(v) - 1), true
			}
		}
	}
	return 0, false
}
