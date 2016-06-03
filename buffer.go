package objects

type buffer struct {
	Channel         chan *Object
	Exit            chan struct{}
	collection      string
	buf             [][]byte
	currentByteSize int
}

func newBuffer(collection string) *buffer {
	return &buffer{
		collection:      collection,
		Channel:         make(chan *Object, 100),
		Exit:            make(chan struct{}, 1),
		buf:             [][]byte{},
		currentByteSize: 0,
	}
}

func (b *buffer) add(x []byte) {
	b.buf = append(b.buf, x)
	b.currentByteSize += len(x)
}

func (b *buffer) size() int {
	return b.currentByteSize
}

func (b *buffer) count() int {
	return len(b.buf)
}

func (b *buffer) reset() {
	b.buf = [][]byte{}
	b.currentByteSize = 0
}
