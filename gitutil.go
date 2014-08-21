/*
Package gitutil provides pure Go access to Git information.
*/
package gitutil

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// RemoteRefs fetches all the refs exposed by a remote repository.
// It returns a map from symbolic ref (e.g. "refs/heads/master") to SHA-1 hash.
// Only HTTP and HTTPS URLs are supported.
func RemoteRefs(client *http.Client, repoURL string) (map[string]string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	// This protocol is documented in Documentation/technical/http-protocol.txt
	// in the Git tree.

	// Fetch /info/refs?service=git-upload-pack
	if !strings.HasSuffix(repoURL, "/") {
		repoURL += "/"
	}
	repoURL += "info/refs?service=git-upload-pack"
	resp, err := client.Get(repoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Sanity check that we're speaking to git.
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad response status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-git-upload-pack-advertisement" {
		return nil, fmt.Errorf("bad response Content-Type %q", ct)
	}

	return parseResponse(body)
}

var (
	startRegexp = regexp.MustCompile(`^[0-9a-f]{4}#`)
	refRegexp   = regexp.MustCompile(`^([0-9a-f]{40}) ([[:print:]]+)(\x00.*)?\n$`)
)

func parseResponse(body []byte) (map[string]string, error) {
	// This is an incomplete parser for Smart Server Response.

	// Check the first five bytes.
	if !startRegexp.Match(body) {
		return nil, fmt.Errorf("first five bytes %.5q are bad", body)
	}

	refs := make(map[string]string)

	br := bytes.NewReader(body)
	for {
		data, flush, err := nextPktLine(br)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bad pkt-line: %v", err)
		}
		if !flush {
			ds := string(data)
			if ds == "# service=git-upload-pack\n" {
				// TODO: only check this for the first line
				continue
			}
			m := refRegexp.FindStringSubmatch(ds)
			if m == nil {
				return nil, fmt.Errorf("bad ref line %q", ds)
			}
			refs[m[2]] = m[1]
			// TODO: deal with capabilities list
		}
	}

	return refs, nil
}

// nextPktLine parses pkt-line (Documentation/technical/protocol-common.txt:51).
// Exactly one of {data,flush} will be non-zero in the non-error case.
func nextPktLine(r io.Reader) (data []byte, flush bool, err error) {
	var nb [4]byte
	if _, err := io.ReadFull(r, nb[:]); err != nil {
		return nil, false, err
	}
	if nb == [4]byte{'0', '0', '0', '0'} {
		// flush-pkt special case
		return nil, true, nil
	}
	n, err := strconv.ParseUint(string(nb[:]), 16, 16)
	if err != nil {
		return nil, false, err
	}
	if n < 4 || n > 65520 {
		return nil, false, fmt.Errorf("bad pkt-len %q (%d)", nb, n)
	}
	data = make([]byte, int(n)-4)
	_, err = io.ReadFull(r, data)
	return data, false, err
}
