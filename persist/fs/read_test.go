// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/m3db/m3db/digest"
	"github.com/m3db/m3db/persist/encoding/msgpack"
	"github.com/m3db/m3db/persist/schema"
	"github.com/m3db/m3db/ts"
	"github.com/m3db/m3x/checked"
	"github.com/m3db/m3x/pool"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testReaderBufferSize = 10
	testWriterBufferSize = 10
)

var (
	testWriterStart = time.Now()
	testBlockSize   = 2 * time.Hour
	testBytesPool   pool.CheckedBytesPool
)

func init() {
	testBytesPool = pool.NewCheckedBytesPool([]pool.Bucket{pool.Bucket{
		Capacity: 1024,
		Count:    10,
	}}, nil, func(s []pool.Bucket) pool.BytesPool {
		return pool.NewBytesPool(s, nil)
	})
	testBytesPool.Init()
}

func newTestReader(t *testing.T, filePathPrefix string) FileSetReader {
	reader, err := NewReader(testBytesPool, NewOptions().
		SetFilePathPrefix(filePathPrefix).
		SetInfoReaderBufferSize(testReaderBufferSize).
		SetDataReaderBufferSize(testReaderBufferSize))
	require.NoError(t, err)
	return reader
}

func bytesRefd(data []byte) checked.Bytes {
	bytes := checked.NewBytes(data, nil)
	bytes.IncRef()
	return bytes
}

// NB(r): This is kind of shitty and brittle, but basically
// msgpack expects a buffered reader, but we can't use a buffered
// reader because we need to know where its up to when we need to grab
// bytes without copying.
//
// This test by its very nature compiling means it implements the
// `bufReader` interface in msgpack decoder library (unless it changes...)
// in which case this needs to be updated.
//
// By it implementing the interface the msgpack decoder actually uses
// the reader directly without creating a buffered reader to wrap it.
// This way we can know actually where its up to and can correctly
// take the right bytes ref address when reading bytes without copying.
type msgpackBufReader interface {
	Read([]byte) (int, error)
	ReadByte() (byte, error)
	UnreadByte() error
}

func TestDecoderStreamImplementsMsgpackBufReader(t *testing.T) {
	r := msgpackBufReader(newDecoderStream())
	assert.NotNil(t, r)
}

// NB(r): todo make a test that gives a decoderStream or something
// looking like a decoder stream to the msgpack library and ensures
// that it doesn't wrap the reader in a bufio.NewReader(...) by analyzing
// the goroutine stack with runtime.Stack()
// ...

func TestReadEmptyIndexUnreadData(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	filePathPrefix := filepath.Join(dir, "")
	defer os.RemoveAll(dir)

	w := newTestWriter(t, filePathPrefix)
	err = w.Open(testNs1ID, testBlockSize, 0, testWriterStart)
	assert.NoError(t, err)
	assert.NoError(t, w.Close())

	r := newTestReader(t, filePathPrefix)
	err = r.Open(testNs1ID, 0, testWriterStart)
	assert.NoError(t, err)

	_, _, _, err = r.Read()
	assert.Error(t, err)

	assert.NoError(t, r.Close())
}

func TestReadDataError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	filePathPrefix := filepath.Join(dir, "")
	defer os.RemoveAll(dir)

	w := newTestWriter(t, filePathPrefix)
	err = w.Open(testNs1ID, testBlockSize, 0, testWriterStart)
	require.NoError(t, err)
	require.NoError(t, w.Write(
		ts.StringID("foo"),
		bytesRefd([]byte{1, 2, 3}),
		digest.Checksum([]byte{1, 2, 3})))
	require.NoError(t, w.Close())

	r := newTestReader(t, filePathPrefix)
	err = r.Open(testNs1ID, 0, testWriterStart)
	assert.NoError(t, err)

	// Close out the dataFd and use a mock to expect an error on next read
	reader := r.(*reader)
	require.NoError(t, munmap(reader.dataMmap))
	require.NoError(t, reader.dataFd.Close())

	mockReader := digest.NewMockReaderWithDigest(ctrl)
	mockReader.EXPECT().Read(gomock.Any()).Return(0, fmt.Errorf("an error"))
	reader.dataReader = mockReader

	_, _, _, err = r.Read()
	assert.Error(t, err)

	// Cleanly close
	require.NoError(t, munmap(reader.indexMmap))
	require.NoError(t, reader.indexFd.Close())
}

func TestReadDataUnexpectedSize(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	filePathPrefix := filepath.Join(dir, "")
	defer os.RemoveAll(dir)

	w := newTestWriter(t, filePathPrefix)
	err = w.Open(testNs1ID, testBlockSize, 0, testWriterStart)
	assert.NoError(t, err)
	dataFile := w.(*writer).dataFdWithDigest.Fd().Name()

	assert.NoError(t, w.Write(
		ts.StringID("foo"),
		bytesRefd([]byte{1, 2, 3}),
		digest.Checksum([]byte{1, 2, 3})))
	assert.NoError(t, w.Close())

	// Truncate one bye
	assert.NoError(t, os.Truncate(dataFile, 1))

	r := newTestReader(t, filePathPrefix)
	err = r.Open(testNs1ID, 0, testWriterStart)
	assert.NoError(t, err)

	_, _, _, err = r.Read()
	assert.Error(t, err)
	assert.Equal(t, errReadNotExpectedSize, err)

	assert.NoError(t, r.Close())
}

func TestReadBadMarker(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	filePathPrefix := filepath.Join(dir, "")
	defer os.RemoveAll(dir)

	w := newTestWriter(t, filePathPrefix)
	err = w.Open(testNs1ID, testBlockSize, 0, testWriterStart)
	assert.NoError(t, err)

	// Copy the marker out
	actualMarker := make([]byte, markerLen)
	assert.Equal(t, markerLen, copy(actualMarker, marker))

	// Mess up the marker
	marker[0] = marker[0] + 1

	assert.NoError(t, w.Write(
		ts.StringID("foo"),
		bytesRefd([]byte{1, 2, 3}),
		digest.Checksum([]byte{1, 2, 3})))

	// Reset the marker
	marker = actualMarker

	assert.NoError(t, w.Close())

	r := newTestReader(t, filePathPrefix)
	err = r.Open(testNs1ID, 0, testWriterStart)
	assert.NoError(t, err)

	_, _, _, err = r.Read()
	assert.Error(t, err)
	assert.Equal(t, errReadMarkerNotFound, err)

	assert.NoError(t, r.Close())
}

func TestReadWrongIdx(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	filePathPrefix := filepath.Join(dir, "")
	defer os.RemoveAll(dir)

	w := newTestWriter(t, filePathPrefix)
	err = w.Open(testNs1ID, testBlockSize, 0, testWriterStart)
	assert.NoError(t, err)

	assert.NoError(t, w.Write(
		ts.StringID("foo"),
		bytesRefd([]byte{1, 2, 3}),
		digest.Checksum([]byte{1, 2, 3})))
	assert.NoError(t, w.Close())

	// Replace the expected idx with 123
	enc := msgpack.NewEncoder()
	entry := schema.IndexEntry{Index: 123}
	require.NoError(t, enc.EncodeIndexEntry(entry))

	shardDir := ShardDirPath(filePathPrefix, testNs1ID, 0)
	indexFilePath := filesetPathFromTime(shardDir, testWriterStart, indexFileSuffix)
	err = ioutil.WriteFile(indexFilePath, enc.Bytes(), defaultNewFileMode)
	require.NoError(t, err)

	r := newTestReader(t, filePathPrefix)
	err = r.Open(testNs1ID, 0, testWriterStart)
	require.NoError(t, err)

	_, _, _, err = r.Read()
	assert.Error(t, err)

	typedErr, ok := err.(ErrReadWrongIdx)
	assert.True(t, ok)
	if ok {
		assert.NotEmpty(t, typedErr.Error())

		// Want 123
		assert.Equal(t, int64(123), typedErr.ExpectedIdx)
		// Got 0
		assert.Equal(t, int64(0), typedErr.ActualIdx)
	}

	assert.NoError(t, r.Close())
}

func TestReadNoCheckpointFile(t *testing.T) {
	filePathPrefix := createTempDir(t)
	defer os.RemoveAll(filePathPrefix)

	w := newTestWriter(t, filePathPrefix)
	shard := uint32(0)
	err := w.Open(testNs1ID, testBlockSize, shard, testWriterStart)
	assert.NoError(t, err)
	assert.NoError(t, w.Close())

	shardDir := ShardDirPath(filePathPrefix, testNs1ID, shard)
	checkpointFile := filesetPathFromTime(shardDir, testWriterStart, checkpointFileSuffix)
	require.True(t, FileExists(checkpointFile))
	os.Remove(checkpointFile)

	r := newTestReader(t, filePathPrefix)
	err = r.Open(testNs1ID, shard, testWriterStart)
	require.Equal(t, errCheckpointFileNotFound, err)
}

func testReadOpen(t *testing.T, fileData map[string][]byte) {
	filePathPrefix := createTempDir(t)
	defer os.RemoveAll(filePathPrefix)

	shard := uint32(0)
	start := time.Unix(1000, 0)
	shardDir := ShardDirPath(filePathPrefix, testNs1ID, shard)

	w := newTestWriter(t, filePathPrefix)
	assert.NoError(t, w.Open(testNs1ID, testBlockSize, uint32(shard), start))

	assert.NoError(t, w.Write(
		ts.StringID("foo"),
		bytesRefd([]byte{0x1}),
		digest.Checksum([]byte{0x1})))
	assert.NoError(t, w.Close())

	for suffix, data := range fileData {
		digestFile := filesetPathFromTime(shardDir, start, suffix)
		fd, err := os.OpenFile(digestFile, os.O_WRONLY|os.O_TRUNC, os.FileMode(0666))
		require.NoError(t, err)
		_, err = fd.Write(data)
		require.NoError(t, err)
		fd.Close()
	}

	r := newTestReader(t, filePathPrefix)
	require.Error(t, r.Open(testNs1ID, shard, time.Unix(1000, 0)))
}

func TestReadOpenDigestOfDigestMismatch(t *testing.T) {
	testReadOpen(
		t,
		map[string][]byte{
			infoFileSuffix:       []byte{0x1},
			indexFileSuffix:      []byte{0x2},
			dataFileSuffix:       []byte{0x3},
			digestFileSuffix:     []byte{0x2, 0x0, 0x2, 0x0, 0x3, 0x0, 0x3, 0x0, 0x4, 0x0, 0x4, 0x0},
			checkpointFileSuffix: []byte{0x12, 0x0, 0x7a, 0x0},
		},
	)
}

func TestReadOpenInfoDigestMismatch(t *testing.T) {
	testReadOpen(
		t,
		map[string][]byte{
			infoFileSuffix:       []byte{0xa},
			indexFileSuffix:      []byte{0x2},
			dataFileSuffix:       []byte{0x3},
			digestFileSuffix:     []byte{0x2, 0x0, 0x2, 0x0, 0x3, 0x0, 0x3, 0x0, 0x4, 0x0, 0x4, 0x0},
			checkpointFileSuffix: []byte{0x13, 0x0, 0x7a, 0x0},
		},
	)
}

func TestReadOpenIndexDigestMismatch(t *testing.T) {
	// Write the correct info digest
	enc := msgpack.NewEncoder()
	require.NoError(t, enc.EncodeIndexInfo(schema.IndexInfo{}))
	b := enc.Bytes()
	di := digest.NewDigest()
	_, err := di.Write(b)
	require.NoError(t, err)

	// Write the wrong index digest
	buf := digest.NewBuffer()
	buf.WriteDigest(di.Sum32())
	digestOfDigest := append(buf, make([]byte, 8)...)
	di.Reset()
	_, err = di.Write(digestOfDigest)
	require.NoError(t, err)
	buf.WriteDigest(di.Sum32())

	testReadOpen(
		t,
		map[string][]byte{
			infoFileSuffix:       b,
			indexFileSuffix:      []byte{0xa},
			dataFileSuffix:       []byte{0x3},
			digestFileSuffix:     digestOfDigest,
			checkpointFileSuffix: buf,
		},
	)
}

func TestReadValidate(t *testing.T) {
	filePathPrefix := createTempDir(t)
	defer os.RemoveAll(filePathPrefix)

	shard := uint32(0)
	start := time.Unix(1000, 0)
	w := newTestWriter(t, filePathPrefix)
	require.NoError(t, w.Open(testNs1ID, testBlockSize, shard, start))

	assert.NoError(t, w.Write(
		ts.StringID("foo"),
		bytesRefd([]byte{0x1}),
		digest.Checksum([]byte{0x1})))
	require.NoError(t, w.Close())

	r := newTestReader(t, filePathPrefix)
	require.NoError(t, r.Open(testNs1ID, shard, start))
	_, _, _, err := r.Read()
	require.NoError(t, err)

	// Mutate expected data checksum to simulate data corruption
	reader := r.(*reader)
	reader.expectedDataDigest = 0
	require.Error(t, r.Validate())

	require.NoError(t, r.Close())
}
