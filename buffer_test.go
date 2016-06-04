package objects

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestBuffer(t *testing.T) {
	suite.Run(t, &BufferTestSuite{})
}

type BufferTestSuite struct {
	suite.Suite
}

func (b *BufferTestSuite) TestNewBuffer() {
	buf := newBuffer("collection")
	b.NotNil(buf)

	b.Equal("collection", buf.collection)
	b.Equal(0, buf.count())
	b.Equal(0, buf.size())
	b.Equal(0, buf.currentByteSize)
	b.Len(buf.buf, 0)
	b.NotNil(buf.Channel)
	b.NotNil(buf.Exit)
}

func (b *BufferTestSuite) TestAddNew() {
	buf := newBuffer("collection")
	b.NotNil(buf)
	json1 := []byte(`{"string": "test", "int": 1}`)
	buf.add(json1)

	b.Equal(1, buf.count())
	b.Equal(len(json1), buf.size())
}

func (b *BufferTestSuite) TestMarshalEmptyArray() {
	buf := newBuffer("collection")
	b.NotNil(buf)
	res := buf.marshalArray()
	b.Equal("[]", string(res))

	v := []map[string]interface{}{}
	b.NoError(json.Unmarshal(res, &v))
	b.Len(v, 0)
}

func (b *BufferTestSuite) TestMarshalSingleArray() {
	buf := newBuffer("collection")
	b.NotNil(buf)
	json1 := []byte(`{"string": "test", "int": 1}`)
	buf.add(json1)

	res := buf.marshalArray()
	b.Equal(`[`+string(json1)+`]`, string(res))

	v := []map[string]interface{}{}
	b.NoError(json.Unmarshal(res, &v))
	b.Len(v, 1)
	b.Equal(v[0]["string"], "test")
	b.Equal(v[0]["int"], float64(1)) // json package uses float64 for all json numbers by default
}

func (b *BufferTestSuite) TestAddMultiple() {
	buf := newBuffer("collection")
	b.NotNil(buf)
	json1 := []byte(`{"string": "test", "int": 1}`)
	buf.add(json1)

	b.Equal(1, buf.count())
	b.Equal(len(json1), buf.size())

	json2 := []byte(`{"string": "test", "int": 46}`)
	buf.add(json2)

	b.Equal(2, buf.count())
	b.Equal(len(json1)+len(json2), buf.size())

	json3 := []byte(`{"string": "test_3", "int": 1000}`)
	buf.add(json3)

	json4 := []byte(`{"string": "test_4", "float": -1.0}`)
	buf.add(json4)

	b.Equal(4, buf.count())
	b.Equal(len(json1)+len(json2)+len(json3)+len(json4), buf.size())
}

func (b *BufferTestSuite) TestAddMultipleReset() {
	buf := newBuffer("collection")
	b.NotNil(buf)
	json1 := []byte(`{"string": "test", "int": 1}`)
	buf.add(json1)

	b.Equal(1, buf.count())
	b.Equal(len(json1), buf.size())

	json2 := []byte(`{"string": "test", "int": 46}`)
	buf.add(json2)

	b.Equal(2, buf.count())
	b.Equal(len(json1)+len(json2), buf.size())

	json3 := []byte(`{"string": "test_3", "int": 1000}`)
	buf.add(json3)

	b.Equal(3, buf.count())
	b.Equal(len(json1)+len(json2)+len(json3), buf.size())

	buf.reset()
	b.Equal(0, buf.count())
	b.Equal(0, buf.size())
	b.Equal(0, buf.currentByteSize)
}

func (b *BufferTestSuite) TestAddMultipleMarshalReset() {
	buf := newBuffer("collection")
	b.NotNil(buf)
	json1 := []byte(`{"string": "test", "int": 1}`)
	buf.add(json1)

	b.Equal(1, buf.count())
	b.Equal(len(json1), buf.size())

	json2 := []byte(`{"string": "test", "int": 46}`)
	buf.add(json2)

	b.Equal(2, buf.count())
	b.Equal(len(json1)+len(json2), buf.size())

	json3 := []byte(`{"string": "test_3", "int": -1.0}`)
	buf.add(json3)

	b.Equal(3, buf.count())
	b.Equal(len(json1)+len(json2)+len(json3), buf.size())

	res := buf.marshalArray()
	b.Equal(`[`+string(json1)+`,`+string(json2)+`,`+string(json3)+`]`, string(res))

	v := []map[string]interface{}{}
	b.NoError(json.Unmarshal(res, &v))
	b.Len(v, 3)
	b.Equal(v[0]["string"], "test")
	b.Equal(v[0]["int"], float64(1))
	b.Equal(v[1]["int"], float64(46))
	b.Equal(v[2]["int"], float64(-1.0))
	b.Equal(v[2]["string"], "test_3")

	buf.reset()
	b.Equal(0, buf.count())
	b.Equal(0, buf.size())
	b.Equal(0, buf.currentByteSize)
}
