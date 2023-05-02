//go:build go1.20
// +build go1.20

package validator

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmptyNames(t *testing.T) {
	var err error

	el := tokenize(t, `<x:>`).(xml.StartElement)
	require.Equal(t, `x:`, el.Name.Local,
		"encoding/xml should tokenize a prefix-only name as a local name")

	err = Validate(bytes.NewBufferString(`<x:>`))
	require.NoError(t, err, "Should not error on start element with no local name")

	err = Validate(bytes.NewBufferString(`</x:>`))
	require.NoError(t, err, "Should not error on end element with no local name")
}

func TestEmptyAttributes(t *testing.T) {
	var err error

	el := tokenize(t, `<Root :="value"/>`).(xml.StartElement)
	require.Equal(t, `:`, el.Attr[0].Name.Local,
		"encoding/xml should tokenize an empty attribute name as a single colon")

	err = Validate(bytes.NewBufferString(`<Root :="value"/>`))
	require.NoError(t, err, "Should not error on input with empty attribute names")

	err = Validate(bytes.NewBufferString(`<Root x:="value"/>`))
	require.NoError(t, err, "Should not error on input with empty attribute local names")

	err = Validate(bytes.NewBufferString(`<Root xmlns="x" xmlns:="y"></Root>`))
	require.NoError(t, err, "Should not error on input with empty xmlns local names")

	validXmlns := `<Root xmlns="http://example.com/"/>`
	require.NoError(t, Validate(bytes.NewBufferString(validXmlns)), "Should pass on input with valid xmlns attributes")
}

func TestDirectives(t *testing.T) {
	var err error

	dir := tokenize(t, `<! x<!-- -->y>`).(xml.Directive)
	require.Equal(t, ` x y`, string(dir), "encoding/xml should replace comments with spaces when tokenizing directives")

	err = Validate(bytes.NewBufferString(
		`<Root>
			<! <<!-- -->!-->"--> " >
			<! ">" <X/>>
		</Root>`))
	require.NoError(t, err, "Should not error on bad directive")

	err = Validate(bytes.NewBufferString(`<Root><! <<!-- -->!-- x --> y></Root>`))
	require.NoError(t, err, "Should not error on bad directive")

	goodDirectives := []string{
		`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`,
		`<! ">" <X/>>`,
		`<!name <!-- comment --><nesting <more nesting>>>`,
	}
	for _, doc := range goodDirectives {
		require.NoError(t, Validate(bytes.NewBufferString(doc)), "Should pass on good directives")
	}
}

func TestValidateAll(t *testing.T) {
	xmlBytes := []byte("<Root>\r\n    <! <<!-- -->!-- x --> y>\r\n    <Element :attr=\"foo\"></x:Element>\r\n</Root>")
	errs := ValidateAll(bytes.NewBuffer(xmlBytes))
	require.Equal(t, 0, len(errs), "Should return zero errors")
}
