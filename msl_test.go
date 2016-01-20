package main

import (
	"bufio"
	"bytes"
	"testing"
)

func TestMultiScanLine(t *testing.T) {

	type Case struct {
		noOfLines int
		input     string
		output    []string
	}
	cs := []Case{
		Case{1, "0\n1\n", []string{"0\n", "1\n"}},
		Case{2, "0\n1\n", []string{"0\n1\n"}},
		Case{2, "0\n1\n2", []string{"0\n1\n", "2"}},
		Case{2, "0000\n1\n2222222\n", []string{"0000\n1\n", "2222222\n"}},
	}

	split := func(c Case) []string {
		s := bufio.NewScanner(bytes.NewBufferString(c.input))
		s.Split(multiScanLinesFactory(c.noOfLines))
		output := []string{}
		for s.Scan() {
			output = append(output, s.Text())
		}
		return output
	}

	for i, c := range cs {
		output := split(c)
		if len(output) != len(c.output) {
			t.Fatalf("%d#: wrong len", i)
		}
		for j, v := range c.output {
			if v != output[j] {
				t.Fatalf("%d#: wrong %d value", i, j)
			}
		}
	}

}
