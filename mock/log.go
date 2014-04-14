// Copyright 2014 ALTOROS
// Licensed under the AGPLv3, see LICENSE file for details.

package mock

import (
	"flag"
	"net/http/httputil"
	"strings"
	"testing"
)

var logFlag *string = flag.String("log.mock", "n", "log mock server requests: none|n, url|u, detail|d")

type severity int

const (
	logNone severity = iota
	logURL
	logDetail
)

func parseLogSeverity(s *string) severity {
	if s == nil || len(*s) == 0 {
		return logNone
	}
	switch (*s)[0] {
	case 'n':
		return logNone
	case 'u':
		return logURL
	case 'd':
		return logDetail
	default:
		return logNone
	}
}

func log() severity {
	return parseLogSeverity(logFlag)
}

func Log(t *testing.T, jj []JournalEntry) {
	for _, j := range jj {
		switch log() {
		case logURL:
			LogURL(t, j)
		case logDetail:
			LogDetail(t, j)
		}
	}
}

func LogURL(t *testing.T, j JournalEntry) {
	t.Log(j.Request.RequestURI)
}

func LogDetail(t *testing.T, j JournalEntry) {
	req := j.Request
	buf, err := httputil.DumpRequest(req, true)
	if err != nil {
		t.Error("Error dumping request:", err)
		return
	}

	t.Log(string(buf))
	t.Log()

	resp := j.Response
	t.Logf("HTTP/%d", resp.Code)
	for header, values := range resp.Header() {
		t.Log(header+":", strings.Join(values, ","))
	}
	t.Log()
	t.Log(resp.Body.String())
	t.Log()
}
