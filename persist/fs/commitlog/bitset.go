// Copyright (c) 2017 Uber Technologies, Inc.
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

package commitlog

const (
	wordSize    = uint(64)
	logWordSize = uint(6) // lg(wordSize)
	wordMask    = wordSize - 1
)

type bitSet interface {
	test(i uint) bool
	set(i uint)
	clearAll()
	getValues() []uint64
}

type bitSetImpl struct {
	values []uint64
}

func newBitSet(size uint) *bitSetImpl {
	return &bitSetImpl{values: make([]uint64, bitSetIndexOf(size)+1)}
}

func (b *bitSetImpl) test(i uint) bool {
	idx := bitSetIndexOf(i)
	if idx >= len(b.values) {
		return false
	}
	return b.values[idx]&(1<<(i&wordMask)) != 0
}

func (b *bitSetImpl) set(i uint) {
	idx := bitSetIndexOf(i)
	currLen := len(b.values)
	if idx >= currLen {
		newValues := make([]uint64, 2*(idx+1))
		copy(newValues, b.values)
		b.values = newValues
	}
	b.values[idx] |= 1 << (i & wordMask)
}

func (b *bitSetImpl) clearAll() {
	for i := range b.values {
		b.values[i] = 0
	}
}

func (b *bitSetImpl) getValues() []uint64 {
	return b.values
}

func bitSetIndexOf(i uint) int {
	return int(i >> logWordSize)
}
