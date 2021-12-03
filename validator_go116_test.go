//+build !go1.17

package validator

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestColonsInLocalNames(t *testing.T) {
	var err error

	err = Validate(bytes.NewBufferString(`<x::Root/>`))
	require.Error(t, err, "Should error on input with colons in the root element's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<x::Root/>`),
		Observed: tokenize(t, `<Root xmlns="x"/>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate(bytes.NewBufferString(`<Root><x::Element></::Element></Root>`))
	require.Error(t, err, "Should error on input with colons in a nested tag's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<x::Element>`),
		Observed: tokenize(t, `<Element xmlns="x">`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate(bytes.NewBufferString(`<Root><Element ::attr="foo"></Element></Root>`))
	require.Error(t, err, "Should error on input with colons in an attribute's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Element ::attr="foo">`),
		Observed: tokenize(t, `<Element attr="foo">`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate(bytes.NewBufferString(`<Root></x::Element></Root>`))
	require.Error(t, err, "Should error on input with colons in an end tag's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `</x::Element>`),
		Observed: tokenize(t, `</Element>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")
}

func TestEmptyNames(t *testing.T) {
	var err error

	err = Validate(bytes.NewBufferString(`<x:>`))
	require.Error(t, err, "Should error on start element with no local name")

	err = Validate(bytes.NewBufferString(`</x:>`))
	require.Error(t, err, "Should error on end element with no local name")
}

func TestEmptyAttributes(t *testing.T) {
	var err error

	err = Validate(bytes.NewBufferString(`<Root :="value"/>`))
	require.Error(t, err, "Should error on input with empty attribute names")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Root :="value"/>`),
		Observed: tokenize(t, `<Root/>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate(bytes.NewBufferString(`<Root x:="value"/>`))
	require.Error(t, err, "Should error on input with empty attribute local names")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Root x:="value"/>`),
		Observed: tokenize(t, `<Root/>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate(bytes.NewBufferString(`<Root xmlns="x" xmlns:="y"></Root>`))
	require.Error(t, err, "Should error on input with empty xmlns local names")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Root xmlns="x" xmlns:="y">`),
		Observed: tokenize(t, `<Root xmlns="x">`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	validXmlns := `<Root xmlns="http://example.com/"/>`
	require.NoError(t, Validate(bytes.NewBufferString(validXmlns)), "Should pass on input with valid xmlns attributes")
}

func TestDirectives(t *testing.T) {
	var err error

	err = Validate(bytes.NewBufferString(
		`<Root>
			<! <<!-- -->!-->"--> " >
			<! ">" <X/>>
		</Root>`))
	require.Error(t, err, "Should error on bad directive")
	require.Equal(t, &xml.SyntaxError{Msg: io.ErrUnexpectedEOF.Error(), Line: 1},
		errors.Unwrap(err), "Round trip should fail with unexpected EOF")

	err = Validate(bytes.NewBufferString(`<Root><! <<!-- -->!-- x --> y></Root>`))
	require.Error(t, err, "Should error on bad directive")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<! <<!-- -->!-- x --> y>`),
		Observed: tokenize(t, `<!  y>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

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
	var err XMLValidationError

	xmlBytes := []byte("<Root>\r\n    <! <<!-- -->!-- x --> y>\r\n    <Element ::attr=\"foo\"></x::Element>\r\n</Root>")
	errs := ValidateAll(bytes.NewBuffer(xmlBytes))
	require.Equal(t, 3, len(errs), "Should return three errors")

	err = errs[0].(XMLValidationError)
	require.Equal(t, int64(2), err.Line, "First error should be on line 2")
	require.Equal(t, int64(5), err.Column, "First error should be on column 5")
	require.Equal(t, []byte(`<! <<!-- -->!-- x --> y>`), xmlBytes[err.Start:err.End], "First error should point to the correct bytes in the original XML")

	err = errs[1].(XMLValidationError)
	require.Equal(t, int64(3), err.Line, "Second error should be on line 3")
	require.Equal(t, int64(5), err.Column, "Second error should be on column 5")
	require.Equal(t, []byte(`<Element ::attr="foo">`), xmlBytes[err.Start:err.End], "Second error should point to the correct bytes in the original XML")

	err = errs[2].(XMLValidationError)
	require.Equal(t, int64(3), err.Line, "Third error should be on line 3")
	require.Equal(t, int64(27), err.Column, "Third error should be on column 27")
	require.Equal(t, []byte(`</x::Element>`), xmlBytes[err.Start:err.End], "Third error should point to the correct bytes in the original XML")
}
