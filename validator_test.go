package validator

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidXML(t *testing.T) {
	docs := []string{
		`<Root></Root>`,
		`<x:Root></x:Root>`,
		`<x:Root xmlns:x="http://example.com/"></x:Root>`,
		`<x:Root xmlns="http://example.com/"></x:Root>`,
		`<Root xmlns="http://example.com/1" xmlns="http://example.com/2"></Root>`,
		`<?xml version="1.0" encoding="EUC-JP"?><Root></Root>`,
		`<Root>text &quot;hello&quot;</Root>`,
		`<Root><![CDATA[text "hello"]]></Root>`,
		`<!-- comment --><Root/>`,
		`<Root xmlns="http://example.com/1" x:attr="y"/>`,
		`<x:Root xmlns="http://example.com/1" x:attr="y" x:attr2="z"/>`,
	}

	for _, doc := range docs {
		require.NoError(t, Validate([]byte(doc)), "Should pass on valid XML documents")
	}
}

func TestColonsInLocalNames(t *testing.T) {
	var err error

	err = Validate([]byte(`<x::Root/>`))
	require.Error(t, err, "Should error on input with colons in the root element's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<x::Root/>`),
		Observed: tokenize(t, `<Root xmlns="x"/>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate([]byte(`<Root><x::Element></::Element></Root>`))
	require.Error(t, err, "Should error on input with colons in a nested tag's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<x::Element>`),
		Observed: tokenize(t, `<Element xmlns="x">`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate([]byte(`<Root><Element ::attr="foo"></Element></Root>`))
	require.Error(t, err, "Should error on input with colons in an attribute's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Element ::attr="foo">`),
		Observed: tokenize(t, `<Element attr="foo">`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate([]byte(`<Root></x::Element></Root>`))
	require.Error(t, err, "Should error on input with colons in an end tag's name")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `</x::Element>`),
		Observed: tokenize(t, `</Element>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")
}

func TestEmptyNames(t *testing.T) {
	var err error

	err = Validate([]byte(`<x:>`))
	require.Error(t, err, "Should error on start element with no local name")

	err = Validate([]byte(`</x:>`))
	require.Error(t, err, "Should error on end element with no local name")
}

func TestEmptyAttributes(t *testing.T) {
	var err error

	err = Validate([]byte(`<Root :="value"/>`))
	require.Error(t, err, "Should error on input with empty attribute names")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Root :="value"/>`),
		Observed: tokenize(t, `<Root/>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate([]byte(`<Root x:="value"/>`))
	require.Error(t, err, "Should error on input with empty attribute local names")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Root x:="value"/>`),
		Observed: tokenize(t, `<Root/>`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	err = Validate([]byte(`<Root xmlns="x" xmlns:="y"></Root>`))
	require.Error(t, err, "Should error on input with empty xmlns local names")
	require.Equal(t, XMLRoundtripError{
		Expected: tokenize(t, `<Root xmlns="x" xmlns:="y">`),
		Observed: tokenize(t, `<Root xmlns="x">`),
	}, errors.Unwrap(err), "Error should contain expected token and mutation")

	validXmlns := `<Root xmlns="http://example.com/"/>`
	require.NoError(t, Validate([]byte(validXmlns)), "Should pass on input with valid xmlns attributes")
}

func TestDirectives(t *testing.T) {
	var err error

	err = Validate([]byte(
		`<Root>
			<! <<!-- -->!-->"--> " >
			<! ">" <X/>>
		</Root>`))
	require.Error(t, err, "Should error on bad directive")
	require.Equal(t, &xml.SyntaxError{Msg: io.ErrUnexpectedEOF.Error(), Line: 1},
		errors.Unwrap(err), "Round trip should fail with unexpected EOF")

	err = Validate([]byte(`<Root><! <<!-- -->!-- x --> y></Root>`))
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
		require.NoError(t, Validate([]byte(doc)), "Should pass on good directives")
	}
}

func TestUnparseableXML(t *testing.T) {
	var err error

	err = Validate([]byte(
		`<Root><!--`))
	require.Error(t, err, "Should error on unclosed comment")
	require.IsType(t, &xml.SyntaxError{}, err, "Error should be an &xml.SyntaxError")

	err = Validate([]byte(
		`<Root>]]></Root>`))
	require.Error(t, err, "Should error on unexpected ']]>' sequence")
	require.IsType(t, &xml.SyntaxError{}, err, "Error should be an &xml.SyntaxError")

	errs := ValidateAll([]byte(
		`<Root ::attr="x">]]><x::Element/></Root>`))
	require.Len(t, errs, 2, "Should return exactly two errors")
	require.Error(t, errs[0], "Should error on bad attribute")
	require.Error(t, errs[1], "Should error on unexpected ']]>' sequence")
	require.IsType(t, &xml.SyntaxError{}, errs[1], "Error should be an &xml.SyntaxError")
}

func TestValidateAll(t *testing.T) {
	var err XMLValidationError

	xmlBytes := []byte("<Root>\r\n    <! <<!-- -->!-- x --> y>\r\n    <Element ::attr=\"foo\"></x::Element>\r\n</Root>")
	errs := ValidateAll(xmlBytes)
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

func TestTokenEquals(t *testing.T) {
	tokens := []xml.Token{
		tokenize(t, `token`),
		tokenize(t, `<!--token-->`),
		tokenize(t, `<!token>`),
		tokenize(t, `</token>`),
		tokenize(t, `<?token?>`),
		tokenize(t, `<token>`),
	}

	for i, token1 := range tokens {
		for j, token2 := range tokens {
			if i != j {
				require.False(t, tokenEquals(token1, token2), fmt.Sprintf("A token of type %T shouldn't equal a token of type %T", token1, token2))
			} else {
				require.True(t, tokenEquals(token1, xml.CopyToken(token2)), "A token should equal a copy of itself")
			}
		}
	}

	nonToken := struct{}{}
	require.False(t, tokenEquals(nonToken, nonToken), "Non-token types should never equal")
}

func TestErrorMessages(t *testing.T) {
	require.Equal(t, "validator: in token starting at 2:16: unexpected EOF",
		XMLValidationError{34, 54, 2, 16, io.ErrUnexpectedEOF}.Error(),
		"Validation error message should match expectation")

	require.Equal(t, "roundtrip error: expected {{ Foo} []}, observed {{ Bar} []}",
		XMLRoundtripError{tokenize(t, `<Foo>`), tokenize(t, `<Bar>`), nil}.Error(),
		"Roundtrip error message with mismatching tokens should match expectation")

	require.Equal(t, "roundtrip error: unexpected overflow after token: bar",
		XMLRoundtripError{tokenize(t, `<Foo>`), tokenize(t, `<Foo>`), []byte(`bar`)}.Error(),
		"Roundtrip error message with overflow should match expectation")
}

func tokenize(t *testing.T, s string) xml.Token {
	decoder := xml.NewDecoder(strings.NewReader(s))
	token, err := decoder.RawToken()
	require.NoError(t, err, "Tokenization should succeed")
	return token
}
