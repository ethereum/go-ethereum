package derivationpath

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type StartingPoint int

const (
	tokenMaster    = 0x6D // char m
	tokenSeparator = 0x2F // char /
	tokenHardened  = 0x27 // char '
	tokenDot       = 0x2E // char .

	hardenedStart = 0x80000000 // 2^31
)

const (
	StartingPointMaster StartingPoint = iota + 1
	StartingPointCurrent
	StartingPointParent
)

type parseFunc = func() error

type decoder struct {
	r                    *strings.Reader
	f                    parseFunc
	pos                  int
	path                 []uint32
	start                StartingPoint
	currentToken         string
	currentTokenHardened bool
}

func newDecoder(path string) *decoder {
	d := &decoder{
		r: strings.NewReader(path),
	}

	d.reset()

	return d
}

func (d *decoder) reset() {
	d.r.Seek(0, io.SeekStart)
	d.pos = 0
	d.start = StartingPointCurrent
	d.f = d.parseStart
	d.path = make([]uint32, 0)
	d.resetCurrentToken()
}

func (d *decoder) resetCurrentToken() {
	d.currentToken = ""
	d.currentTokenHardened = false
}

func (d *decoder) parse() (StartingPoint, []uint32, error) {
	for {
		err := d.f()
		if err != nil {
			if err == io.EOF {
				err = nil
			} else {
				err = fmt.Errorf("at position %d, %s", d.pos, err.Error())
			}

			return d.start, d.path, err
		}
	}

	return d.start, d.path, nil
}

func (d *decoder) readByte() (byte, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return b, err
	}

	d.pos++

	return b, nil
}

func (d *decoder) unreadByte() error {
	err := d.r.UnreadByte()
	if err != nil {
		return err
	}

	d.pos--

	return nil
}

func (d *decoder) parseStart() error {
	b, err := d.readByte()
	if err != nil {
		return err
	}

	if b == tokenMaster {
		d.start = StartingPointMaster
		d.f = d.parseSeparator
		return nil
	}

	if b == tokenDot {
		b2, err := d.readByte()
		if err != nil {
			return err
		}

		if b2 == tokenDot {
			d.f = d.parseSeparator
			d.start = StartingPointParent
			return nil
		}

		d.f = d.parseSeparator
		d.start = StartingPointCurrent
		return d.unreadByte()
	}

	d.f = d.parseSegment

	return d.unreadByte()
}

func (d *decoder) saveSegment() error {
	if len(d.currentToken) > 0 {
		i, err := strconv.ParseUint(d.currentToken, 10, 32)
		if err != nil {
			return err
		}

		if i >= hardenedStart {
			d.pos -= len(d.currentToken) - 1
			return fmt.Errorf("index must be lower than 2^31, got %d", i)
		}

		if d.currentTokenHardened {
			i += hardenedStart
		}

		d.path = append(d.path, uint32(i))
	}

	d.f = d.parseSegment
	d.resetCurrentToken()

	return nil
}

func (d *decoder) parseSeparator() error {
	b, err := d.readByte()
	if err != nil {
		return err
	}

	if b == tokenSeparator {
		return d.saveSegment()
	}

	return fmt.Errorf("expected %s, got %s", string(tokenSeparator), string(b))
}

func (d *decoder) parseSegment() error {
	b, err := d.readByte()
	if err == io.EOF {
		if len(d.currentToken) == 0 {
			return fmt.Errorf("expected number, got EOF")
		}

		if newErr := d.saveSegment(); newErr != nil {
			return newErr
		}

		return err
	}

	if err != nil {
		return err
	}

	if len(d.currentToken) > 0 && b == tokenSeparator {
		return d.saveSegment()
	}

	if len(d.currentToken) > 0 && b == tokenHardened {
		d.currentTokenHardened = true
		d.f = d.parseSeparator
		return nil
	}

	if b < 0x30 || b > 0x39 {
		return fmt.Errorf("expected number, got %s", string(b))
	}

	d.currentToken = fmt.Sprintf("%s%s", d.currentToken, string(b))

	return nil
}

func Decode(str string) (StartingPoint, []uint32, error) {
	d := newDecoder(str)
	return d.parse()
}
