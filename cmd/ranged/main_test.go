package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func cleanup(oldArgs []string) {
	os.Args = oldArgs
	url = ""
}

func TestCheckArgs(t *testing.T) {
	assert := assert.New(t)
	var args = []string{"foo", "bar"}
	assert.Nil(checkArgs(args))
}

func TestCheckArgsError(t *testing.T) {
	assert := assert.New(t)
	err := checkArgs([]string{})
	assert.NotNil(err)
}

func TestGetInput(t *testing.T) {
	assert := assert.New(t)
	oldArgs := os.Args
	defer cleanup(oldArgs)

	os.Args = []string{"ranged", "-url=http://www.foo.com"}
	flag.Parse()
	assert.Equal("http://www.foo.com", url)
}

func TestGetInputShorthand(t *testing.T) {
	assert := assert.New(t)
	oldArgs := os.Args
	defer cleanup(oldArgs)

	os.Args = []string{"ranged", "-u=http://www.foo.com"}
	flag.Parse()
	assert.Equal("http://www.foo.com", url)
}
