package validator

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flateIt takes and input string, compresses it using flate, and returns a flate.Reader() of the compressed content
func flateIt(t *testing.T, input string) io.Reader {
	t.Helper()

	var zipped bytes.Buffer
	w, err := flate.NewWriter(&zipped, flate.DefaultCompression)
	require.NoError(t, err)

	w.Write([]byte(input))
	w.Close()

	return flate.NewReader(&zipped)
}

func TestValidateZippedReader(t *testing.T) {
	// wrap an innocuous "<foo></foo>" XML payload in a flate.Reader :
	zipped := flateIt(t, `<foo></foo>`)

	// Validate should not trigger an error on that Reader :
	err := Validate(zipped)
	assert.NoError(t, err)
}
