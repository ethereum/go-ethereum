package queue

import (
	"encoding/binary"
	"log"
	"time"
)

const (
	// Number of bytes used to keep information about entry size
	headerEntrySize = 4
	// Bytes before left margin are not used. Zero index means element does not exist in queue, useful while reading slice from index
	leftMarginIndex = 1
	// Minimum empty blob size in bytes. Empty blob fills space between tail and head in additional memory allocation.
	// It keeps entries indexes unchanged
	minimumEmptyBlobSize = 32 + headerEntrySize
)

var (
	errEmptyQueue       = &queueError{"Empty queue"}
	errInvalidIndex     = &queueError{"Index must be greater than zero. Invalid index."}
	errIndexOutOfBounds = &queueError{"Index out of range"}
)

// BytesQueue is a non-thread safe queue type of fifo based on bytes array.
// For every push operation index of entry is returned. It can be used to read the entry later
type BytesQueue struct {
	array           []byte
	capacity        int
	maxCapacity     int
	head            int
	tail            int
	count           int
	rightMargin     int
	headerBuffer    []byte
	verbose         bool
	initialCapacity int
}

type queueError struct {
	message string
}

// NewBytesQueue initialize new bytes queue.
// Initial capacity is used in bytes array allocation
// When verbose flag is set then information about memory allocation are printed
func NewBytesQueue(initialCapacity int, maxCapacity int, verbose bool) *BytesQueue {
	return &BytesQueue{
		array:           make([]byte, initialCapacity),
		capacity:        initialCapacity,
		maxCapacity:     maxCapacity,
		headerBuffer:    make([]byte, headerEntrySize),
		tail:            leftMarginIndex,
		head:            leftMarginIndex,
		rightMargin:     leftMarginIndex,
		verbose:         verbose,
		initialCapacity: initialCapacity,
	}
}

// Reset removes all entries from queue
func (q *BytesQueue) Reset() {
	// Just reset indexes
	q.tail = leftMarginIndex
	q.head = leftMarginIndex
	q.rightMargin = leftMarginIndex
	q.count = 0
}

// Push copies entry at the end of queue and moves tail pointer. Allocates more space if needed.
// Returns index for pushed data or error if maximum size queue limit is reached.
func (q *BytesQueue) Push(data []byte) (int, error) {
	dataLen := len(data)

	if q.availableSpaceAfterTail() < dataLen+headerEntrySize {
		if q.availableSpaceBeforeHead() >= dataLen+headerEntrySize {
			q.tail = leftMarginIndex
		} else if q.capacity+headerEntrySize+dataLen >= q.maxCapacity && q.maxCapacity > 0 {
			return -1, &queueError{"Full queue. Maximum size limit reached."}
		} else {
			q.allocateAdditionalMemory(dataLen + headerEntrySize)
		}
	}

	index := q.tail

	q.push(data, dataLen)

	return index, nil
}

func (q *BytesQueue) allocateAdditionalMemory(minimum int) {
	start := time.Now()
	if q.capacity < minimum {
		q.capacity += minimum
	}
	q.capacity = q.capacity * 2
	if q.capacity > q.maxCapacity && q.maxCapacity > 0 {
		q.capacity = q.maxCapacity
	}

	oldArray := q.array
	q.array = make([]byte, q.capacity)

	if leftMarginIndex != q.rightMargin {
		copy(q.array, oldArray[:q.rightMargin])

		if q.tail < q.head {
			emptyBlobLen := q.head - q.tail - headerEntrySize
			q.push(make([]byte, emptyBlobLen), emptyBlobLen)
			q.head = leftMarginIndex
			q.tail = q.rightMargin
		}
	}

	if q.verbose {
		log.Printf("Allocated new queue in %s; Capacity: %d \n", time.Since(start), q.capacity)
	}
}

func (q *BytesQueue) push(data []byte, len int) {
	binary.LittleEndian.PutUint32(q.headerBuffer, uint32(len))
	q.copy(q.headerBuffer, headerEntrySize)

	q.copy(data, len)

	if q.tail > q.head {
		q.rightMargin = q.tail
	}

	q.count++
}

func (q *BytesQueue) copy(data []byte, len int) {
	q.tail += copy(q.array[q.tail:], data[:len])
}

// Pop reads the oldest entry from queue and moves head pointer to the next one
func (q *BytesQueue) Pop() ([]byte, error) {
	data, size, err := q.peek(q.head)
	if err != nil {
		return nil, err
	}

	q.head += headerEntrySize + size
	q.count--

	if q.head == q.rightMargin {
		q.head = leftMarginIndex
		if q.tail == q.rightMargin {
			q.tail = leftMarginIndex
		}
		q.rightMargin = q.tail
	}

	return data, nil
}

// Peek reads the oldest entry from list without moving head pointer
func (q *BytesQueue) Peek() ([]byte, error) {
	data, _, err := q.peek(q.head)
	return data, err
}

// Get reads entry from index
func (q *BytesQueue) Get(index int) ([]byte, error) {
	data, _, err := q.peek(index)
	return data, err
}

// CheckGet checks if an entry can be read from index
func (q *BytesQueue) CheckGet(index int) error {
	return q.peekCheckErr(index)
}

// Capacity returns number of allocated bytes for queue
func (q *BytesQueue) Capacity() int {
	return q.capacity
}

// Len returns number of entries kept in queue
func (q *BytesQueue) Len() int {
	return q.count
}

// Error returns error message
func (e *queueError) Error() string {
	return e.message
}

// peekCheckErr is identical to peek, but does not actually return any data
func (q *BytesQueue) peekCheckErr(index int) error {

	if q.count == 0 {
		return errEmptyQueue
	}

	if index <= 0 {
		return errInvalidIndex
	}

	if index+headerEntrySize >= len(q.array) {
		return errIndexOutOfBounds
	}
	return nil
}

func (q *BytesQueue) peek(index int) ([]byte, int, error) {

	if q.count == 0 {
		return nil, 0, errEmptyQueue
	}

	if index <= 0 {
		return nil, 0, errInvalidIndex
	}

	if index+headerEntrySize >= len(q.array) {
		return nil, 0, errIndexOutOfBounds
	}

	blockSize := int(binary.LittleEndian.Uint32(q.array[index : index+headerEntrySize]))
	return q.array[index+headerEntrySize : index+headerEntrySize+blockSize], blockSize, nil
}

func (q *BytesQueue) availableSpaceAfterTail() int {
	if q.tail >= q.head {
		return q.capacity - q.tail
	}
	return q.head - q.tail - minimumEmptyBlobSize
}

func (q *BytesQueue) availableSpaceBeforeHead() int {
	if q.tail >= q.head {
		return q.head - leftMarginIndex - minimumEmptyBlobSize
	}
	return q.head - q.tail - minimumEmptyBlobSize
}
