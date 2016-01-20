package main

import (
	"bufio"
	"bytes"
)

func multiScanLinesFactory(noOfLines int) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		return multiScanLines(data, atEOF, noOfLines)
	}
}

func multiScanLines(data []byte, atEOF bool, noOfLines int) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	window := data[:]
	var hit, pos int
	for hit = 0; hit < noOfLines; hit++ {
		pos = bytes.IndexByte(window, '\n')
		if pos < 0 {
			break
		}
		advance += pos + 1
		window = data[advance:]
	}
	if hit == noOfLines {
		return advance, data[0:advance], nil
	}

	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
