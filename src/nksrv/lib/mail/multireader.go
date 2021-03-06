package mail

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"nksrv/lib/utils/text/bufreader"
)

type PartReader struct {
	*bufreader.BufReader // current part reader

	br    *bufreader.BufReader // underlying reader
	extbr bool                 // whether br is external

	n                int   // amount of readable data
	err              error // queued error
	rpart            int   // how much of current part was read
	dashBoundaryDash []byte
	dashBoundary     []byte
	nlDashBoundary   []byte
	nl               []byte // may be \r\n (the default) or \n
	partsRead        int
}

type wrapUnderlying struct {
	*PartReader
}

func (w wrapUnderlying) Read(b []byte) (int, error) {
	return w.read(b)
}

func NewPartReader(r io.Reader, boundary string) *PartReader {
	b := []byte("\r\n--" + boundary + "--")

	br, extbr := r.(*bufreader.BufReader)
	/* idk what should be actual limit but this should work */
	if br == nil || br.Size() < 256 {
		extbr = false
		br = obtainBufReader(r)
	}

	return &PartReader{
		br:               br,
		extbr:            extbr,
		dashBoundaryDash: b[2:],
		dashBoundary:     b[2 : len(b)-2],
		nlDashBoundary:   b[:len(b)-2],
		nl:               b[:2],
	}
}

// returns nil incase next part can be read
// returns io.EOF if terminated
// returns other error on read problem
func (pr *PartReader) nextPart() error {
	// terminate current
	cr := pr.BufReader
	if cr != nil {
		cr.Discard(-1)
	}
	br := pr.br
	truncated := false
	expectNewPart := false
	for {
		b := br.Buffered()
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			if pr.err != nil {
				if pr.err == io.EOF && (truncated || !pr.checkPartEndEOF(br.Buffered())) {
					return io.ErrUnexpectedEOF
				}
				return pr.err
			}
			// check if we can read more
			if br.Capacity() == 0 {
				// cant read more, try compact
				if br.Size() > len(b) {
					// do compaction
					br.CompactBuffer()
				} else {
					// cant compact, its just too big -- dont process it
					br.Discard(len(b))
					truncated = true
				}
			}
			_, pr.err = br.FillBufferAtleast(1)
			// check error. if error that means buffer don't have new data
			if pr.err != nil {
				// maybe we have reached ending without [\r]\n?
				// if not, EOF is unexpected
				if pr.err == io.EOF && (truncated || !pr.checkPartEndEOF(br.Buffered())) {
					return io.ErrUnexpectedEOF
				}
				return pr.err
			}
			continue
		}

		line := b[:i+1]
		// we have line of some sort, check if its boundary
		if !truncated && bytes.HasPrefix(line, pr.dashBoundary) {
			// some sort of boundary maybe
			ending := false
			blen := len(pr.dashBoundary)
			if len(line) <= blen+2 || line[blen] != '-' || line[blen+1] != '-' {
				line = line[blen:]
			} else {
				line = line[blen+2:]
				ending = true
			}
			line = skipWS(line)
			if pr.partsRead == 0 && len(line) == 1 && line[0] == '\n' {
				// adopt to \n
				pr.nl = pr.nl[1:]
				pr.nlDashBoundary = pr.nlDashBoundary[1:]
			}
			if bytes.Equal(line, pr.nl) {
				if !ending {
					br.Discard(i + 1)
					pr.partsRead++
					pr.rpart = 0
					break
				} else {
					return io.EOF
				}
			}
		}
		if expectNewPart {
			return fmt.Errorf("was expecting new part, got line %q", line)
		}
		if pr.partsRead == 0 {
			// skip everything before first part
			br.Discard(i + 1)
			continue
		}
		if !truncated && bytes.Equal(line, pr.nl) {
			// newline after last part just before dashBoundary
			expectNewPart = true
			br.Discard(i + 1)
			continue
		}
		return fmt.Errorf("truncated line or unexpected line %q", line)
	}

	if cr == nil {
		cr = obtainBufReader(wrapUnderlying{pr})
		pr.BufReader = cr
	}
	cr.ResetErr()
	return nil
}

func (pr *PartReader) NextPart() (e error) {
	if pr.br == nil {
		return pr.err
	}
	e = pr.nextPart()
	if e != nil {
		pr.Close()
		pr.err = e
	}
	return
}

func (pr *PartReader) Close() error {
	cr := pr.BufReader
	if cr != nil {
		dropBufReader(cr)
		pr.BufReader = nil
	}
	br := pr.br
	if br != nil {
		if !pr.extbr {
			dropBufReader(br)
		}
		pr.br = nil
	}
	return nil
}

func skipWS(b []byte) []byte {
	for len(b) != 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	return b
}

func (pr *PartReader) checkPartEndEOF(line []byte) bool {
	if !bytes.HasPrefix(line, pr.dashBoundaryDash) {
		return false
	}
	line = line[len(pr.dashBoundaryDash):]
	line = skipWS(line)
	return len(line) == 0
}

func (pr *PartReader) ReadHeaders(headlimit int) (H HeaderMap, e error) {
	return limitedReadHeadersFromExisting(pr.BufReader, headlimit)
}

func (pr *PartReader) read(b []byte) (n int, e error) {
	br := pr.br
	for pr.n == 0 {
		// try looking in current buffer first
		e = pr.checkReadable()
		if pr.n == 0 {
			// still nothing, we'll need to read first
			// check returned error on its own?
			if e != nil {
				return
			}
			// we erred and cant read more?
			if pr.err != nil {
				if pr.err == io.EOF {
					e = io.ErrUnexpectedEOF
				} else {
					e = pr.err
				}
				return
			}
			// read more
			if br.Capacity() == 0 {
				// cant read more, but can we fix this?
				if br.Size() > len(br.Buffered()) {
					// do compaction
					br.CompactBuffer()
				} else {
					// cant compact, too big. this shouldnt really happen
					return n, errors.New("too long boundary line")
				}
			}
			//fmt.Fprintf(os.Stderr, "pr.n = 0 so we are refilling\n")
			//var xxxx int
			_, pr.err = br.FillBufferAtleast(1)
			//fmt.Fprintf(os.Stderr, "refilled, (%d,%v)\n", xxxx, pr.err)
		}
	}
	w := len(b)
	if w > pr.n {
		// clamp to what we have
		w = pr.n
	}
	n, _ = br.Read(b[:w])
	pr.rpart += n
	pr.n -= n
	if pr.n != 0 {
		//fmt.Fprintf(os.Stderr, "pr.n(%d) and we e = nil\n", pr.n)
		// if we're able to return more data, don't prematurely err
		e = nil
	}
	return
}

func (pr *PartReader) checkReadable() error {
	b := pr.br.Buffered()
	if pr.rpart == 0 {
		// begining of current part -- check for boundary
		blen := len(pr.dashBoundary)
		if len(b) >= blen {
			if bytes.Equal(b[:blen], pr.dashBoundary) {
				// first part matched
				switch pr.checkAfterPrefix(b[blen:]) {
				case +1:
					// it did match, signal EOF for this read
					return io.EOF
				case 0:
					// not enough data to tell
					return nil
				case -1:
					// no match, add these bytes
					pr.n += blen
					return nil
				}
			}
		} else {
			if bytes.Equal(b, pr.dashBoundary[:len(b)]) {
				// not enough data
				return nil
			}
		}
	}
	// is there nlDashBoundary somewhere in there?
	if i := bytes.Index(b, pr.nlDashBoundary); i >= 0 {
		pr.n += i
		switch pr.checkAfterPrefix(b[i+len(pr.nlDashBoundary):]) {
		case +1:
			return io.EOF
		case 0:
			return nil
		case -1:
			pr.n += len(pr.nlDashBoundary)
			return nil
		}
	}
	// current buffer is start of nlDashBoundary?
	if bytes.HasPrefix(pr.nlDashBoundary, b) {
		return nil
	}
	// slow path: find begining of nlDashBoundary
	// we have already checked for nlDashBoundary itself, so we can search for last occurence now
	if i := bytes.LastIndexByte(b, pr.nl[0]); i >= 0 && bytes.HasPrefix(pr.nlDashBoundary, b[i:]) {
		pr.n += i
		return nil
	}
	// nothing relevant found, so just skip it
	pr.n += len(b)
	return nil
}

// +1 - positive complete match
//  0 - not enough data to tell
// -1 - negative complete match
func (pr *PartReader) checkAfterPrefix(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	endmark := false
	if b[0] == '-' {
		if len(b) == 1 {
			return 0
		}
		if b[1] == '-' {
			endmark = true
			b = b[2:]
		} else {
			return -1
		}
	}
	b = skipWS(b)
	if len(b) == 0 {
		if endmark && pr.err == io.EOF {
			return +1
		}
		return 0
	}
	if len(b) < len(pr.nl) {
		return 0
	}
	if bytes.Equal(b[:len(pr.nl)], pr.nl) {
		return +1
	} else {
		return -1
	}
}
