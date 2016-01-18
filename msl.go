package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

var tokens = []string{}

const noOfLines = 3

func MultiScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanLines(data, atEOF)
	if err != nil {
		return advance, token, err
	}
	if token != nil {
		tokens = append(tokens, string(token))
	}

	if len(tokens) > noOfLines {
		token = []byte(strings.Join(tokens, "\n"))
		tokens = []string{}
	}
	return advance, token, err
}

func testIt() {

	data := bytes.NewBufferString(`aaa
bbb
ccc
ddd
eee
fff
ggg
hhh
iii
`)
	s := bufio.NewScanner(data)
	s.Split(MultiScanLines)
	// s.Split(bufio.ScanLines)
	for s.Scan() {
		println("--------------")
		fmt.Printf("TEXT = %+v\n", s.Text())
	}
	fmt.Printf("s.Eee = %+v\n", s.Err())

}
