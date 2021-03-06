package nntp

import (
	"errors"
	"fmt"
	"io"

	"nksrv/lib/mail"
	au "nksrv/lib/utils/text/asciiutils"
	"nksrv/lib/utils/text/bufreader"
)

func (c *NNTPPuller) readDotLine(dr *bufreader.DotReader) ([]byte, error) {
	i := 0
	for {
		b, e := dr.ReadByte()
		if e != nil {
			return c.inbuf[:i], e
		}
		if b == '\n' {
			return c.inbuf[:i], nil
		}
		if i >= len(c.inbuf) {
			return c.inbuf[:i], errTooLargeResponse
		}
		c.inbuf[i] = b
		i++
	}
}

func (c *NNTPPuller) readOnlyNewsgroup(
	dr *bufreader.DotReader) ([]byte, error) {

	i := 0
	end := 0
	for {
		b, e := dr.ReadByte()
		if e != nil {
			return c.inbuf[:end], e
		}
		if b == '\n' {
			if end == 0 {
				end = i
			}
			if end == 0 || !FullValidGroupSlice(c.inbuf[:end]) {
				return nil, fmt.Errorf("bad group %q", c.inbuf[:end])
			}
			return c.inbuf[:end], nil
		}
		if end == 0 {
			if b == ' ' || b == '\t' {
				end = i
				continue
			}
			if i >= len(c.inbuf) {
				return nil, errTooLargeResponse
			}
			c.inbuf[i] = b
			i++
		}
	}
}

func parseListActiveLine(
	line []byte) (name []byte, hiwm, lowm uint64, status []byte, err error) {

	i := 0
	skipWS := func() {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
	}
	skipNonWS := func() {
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
	}

	//skipWS()
	s := i
	skipNonWS()
	if s >= i || !FullValidGroupSlice(line[s:i]) {
		err = fmt.Errorf("bad group %q", line[s:i])
		return
	}
	name = line[s:i]

	skipWS()
	s = i
	skipNonWS()
	if s >= i || !isNumberSlice(line[s:i]) {
		err = fmt.Errorf("bad hiwm %q", line[s:i])
		return
	}
	hiwm = stoi64(line[s:i])

	skipWS()
	s = i
	skipNonWS()
	if s >= i || !isNumberSlice(line[s:i]) {
		err = fmt.Errorf("bad lowm %q", line[s:i])
		return
	}
	lowm = stoi64(line[s:i])

	skipWS()
	s = i
	skipNonWS()
	// can be empty I guess... I don't see why not
	status = line[s:i]

	// treat any extra as error
	skipWS()
	if i < len(line) {
		err = fmt.Errorf("unknown extra data: %q", line[i:])
		return
	}

	return
}

func (c *NNTPPuller) getOverLineInfo(
	dr *bufreader.DotReader) (
	id uint64, msgid, ref TFullMsgID, err error, fatal bool) {

	i := 0
	nomore := false
	eatField := func() (field []byte, err error) {
		if nomore {
			return
		}
		s := i
		for {
			b, e := dr.ReadByte()
			if e != nil {
				if e != io.EOF {
					fatal = true
				}
				err = e
				return
			}
			if b == '\n' {
				field = c.inbuf[s:i]
				nomore = true
				return
			}
			if b == '\t' {
				field = c.inbuf[s:i]
				return
			}
			if i >= len(c.inbuf) {
				err = errTooLargeResponse
				return
			}
			c.inbuf[i] = b
			i++
		}
	}
	ignoreField := func() (err error) {
		if nomore {
			return
		}
		for {
			b, e := dr.ReadByte()
			if e != nil {
				if e != io.EOF {
					fatal = true
				}
				err = e
				return
			}
			if b == '\n' {
				nomore = true
				return
			}
			if b == '\t' {
				return
			}
		}
	}

	defer func() {
		if !nomore {
			for {
				b, e := dr.ReadByte()
				if e != nil || b == '\n' {
					if e != nil && err == nil {
						err = e
					}
					if e != nil && e != io.EOF {
						fatal = true
					}
					return
				}
			}
		}
	}()

	// {RFC 2980}
	// (article number goes before these, ofc)
	// The sequence of fields must be in this order:
	// subject, author, date, message-id, references,
	// byte count, and line count.

	// number
	snum, err := eatField()
	if err != nil || nomore {
		return
	}
	snum = au.TrimWSBytes(snum)
	if len(snum) == 0 || !isNumberSlice(snum) {
		err = fmt.Errorf("bad id %q", snum)
		return
	}
	id = stoi64(snum)
	// subject, author, date
	for xx := 0; xx < 3; xx++ {
		err = ignoreField()
		if err != nil {
			return
		}
		if nomore {
			err = errors.New("wanted more fields")
			return
		}
	}
	// message-id
	smsgid, err := eatField()
	if err != nil {
		return
	}
	smsgid = au.TrimWSBytes(smsgid)
	msgid = TFullMsgID(smsgid)
	if !ValidMessageID(msgid) {
		err = fmt.Errorf("invalid msg-id %q", smsgid)
		return
	}
	// references
	xref, err := eatField()
	if err != nil {
		return
	}
	ref = TFullMsgID(unsafeStrToBytes(
		string(mail.ExtractFirstValidReference(unsafeBytesToStr(xref)))))

	return
}

func (c *NNTPPuller) eatHdrMsgIDLine(
	dr *bufreader.DotReader) (
	id uint64, msgid TFullMsgID, err error) {

	line, err := c.readDotLine(dr)
	if err != nil {
		return
	}

	//c.log.LogPrintf(DEBUG, "eatHdrMsgIDLine line: %q", line)

	i := 0
	skipWS := func() {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
	}
	skipNonWS := func() {
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
	}

	skipWS()
	s := i
	skipNonWS()
	if s >= i {
		// empty line
		return
	}
	snum := line[s:i]
	// {RFC 2980}
	if au.EqualFoldString(unsafeBytesToStr(snum), "(none)") {
		return
	}
	if !isNumberSlice(snum) {
		err = fmt.Errorf("bad id %q", snum)
		return
	}
	id = stoi64(snum)

	skipWS()
	s = i
	skipNonWS()
	msgid = TFullMsgID(line[s:i])
	if !ValidMessageID(msgid) {
		err = fmt.Errorf("invalid msg-id %q", line[s:i])
		return
	}
	skipWS()
	if i < len(line) {
		err = errors.New("extra data in HDR output")
		return
	}

	return
}

func (c *NNTPPuller) parseGroupResponse(
	rest []byte) (num, lo, hi uint64, err error) {

	defer func() {
		c.args = c.args[:0]
	}()

	c.args, _ = parseResponseArguments(rest, 4, c.args[:0])
	if len(c.args) < 3 ||
		!isNumberSlice(c.args[0]) ||
		!isNumberSlice(c.args[1]) ||
		!isNumberSlice(c.args[2]) {

		err = fmt.Errorf(
			"bad successful group response %q",
			au.TrimWSBytes(rest))
		return
	}

	num = stoi64(c.args[0])
	lo = stoi64(c.args[1])
	hi = stoi64(c.args[2])
	return
}
