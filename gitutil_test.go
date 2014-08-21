package gitutil

import (
	"reflect"
	"strings"
	"testing"
)

// copied from Documentation/technical/http-protocol.txt
const sampleResp = `001e# service=git-upload-pack
004895dcfa3633004da0049d3d0fa03f80589cbcaf31 refs/heads/maint` + "\x00" + `multi_ack
003fd049f6c27a2244e12041955e262a404c7faba355 refs/heads/master
003c2cb58b79488a98d2721cea644875a8dd0026b115 refs/tags/v1.0
003fa3c2e2402b99163d1d59756e5f207ae21cccba4c refs/tags/v1.0^{}
`

func TestParseResponse(t *testing.T) {
	m, err := parseResponse([]byte(sampleResp))
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	want := map[string]string{
		"refs/heads/maint":  "95dcfa3633004da0049d3d0fa03f80589cbcaf31",
		"refs/heads/master": "d049f6c27a2244e12041955e262a404c7faba355",
		"refs/tags/v1.0":    "2cb58b79488a98d2721cea644875a8dd0026b115",
		"refs/tags/v1.0^{}": "a3c2e2402b99163d1d59756e5f207ae21cccba4c",
	}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("Response mismatch.\n got %v\nwant %v", m, want)
	}
}

func TestNextPktLine(t *testing.T) {
	tests := []struct {
		in    string
		data  string
		flush bool
	}{
		{"0000", "", true},
		{"0006a\n", "a\n", false},
		{"0005a", "a", false},
		{"000bfoobar\n", "foobar\n", false},
		{"0004", "", false},

		{"000bfoobar\nxxxx", "foobar\n", false},
	}
	for _, test := range tests {
		data, flush, err := nextPktLine(strings.NewReader(test.in))
		if err != nil {
			t.Errorf("%q failed: %v", test.in, err)
			continue
		}
		d := string(data)
		if d != test.data || flush != test.flush {
			t.Errorf("%q parsed as (%q, %t), want (%q, %t)", test.in, d, flush, test.data, test.flush)
		}
	}
}
