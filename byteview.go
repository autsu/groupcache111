package groupcache

// ByteView 保证了数据的只读
type ByteView struct {
	b []byte
}

func (b *ByteView) Len() int64 {
	return int64(len(b.b))
}

func (b *ByteView) String() string {
	return string(b.b)
}

func (b *ByteView) ByteSlice() []byte {
	return cloneBytes(b.b)
}

func cloneBytes(b []byte) []byte {
	bb := make([]byte, len(b), len(b))
	copy(bb, b)
	return bb
}
