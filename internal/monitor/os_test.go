package monitor

import (
	"strings"
	"testing"
)

func TestParseOSRelease(t *testing.T) {
	for _, test := range []struct{ name, content, want string }{
		{name: "pretty", content: "NAME=Ubuntu\nPRETTY_NAME=\"Ubuntu 24.04 LTS\"", want: "Ubuntu 24.04 LTS"},
		{name: "fallback", content: "NAME='Debian GNU/Linux'\nVERSION_ID=13", want: "Debian GNU/Linux 13"},
		{name: "rolling", content: "NAME='Arch Linux'", want: "Arch Linux"},
		{name: "empty", want: "Linux"},
		{name: "escape", content: "PRETTY_NAME=\"Test \\\"OS\\\" \\$HOME\"", want: "Test \"OS\" $HOME"},
		{name: "duplicate", content: "PRETTY_NAME=First\nPRETTY_NAME=Second", want: "Second"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOSRelease(strings.NewReader(test.content))
			if err != nil || got != test.want {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
		})
	}
	if _, err := parseOSRelease(strings.NewReader("PRETTY_NAME='unfinished")); err == nil {
		t.Fatal("accepted malformed value")
	}
}
