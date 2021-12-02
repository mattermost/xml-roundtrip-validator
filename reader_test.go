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

	_, err = w.Write([]byte(input))
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	return flate.NewReader(&zipped)
}

func TestValidateZippedReader(t *testing.T) {
	// wrap an innocuous "<foo></foo>" XML payload in a flate.Reader :
	zipped := flateIt(t, `<foo></foo>`)

	// Validate should not trigger an error on that Reader :
	err := Validate(zipped)
	assert.NoError(t, err, "Should not error on a valid XML document")

	// an invalid document should still error :
	zipped = flateIt(t, `<Root>]]></Root>`)

	err = Validate(zipped)
	assert.Error(t, err, "Should error on an invalid XML document")
}
