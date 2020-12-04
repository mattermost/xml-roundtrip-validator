# Security Advisory: XML element instability in Go's `encoding/xml`

<table>
  <tr><td>Affected component</td><td>Package <code>encoding/xml</code> in Go</td></tr>
  <tr><td>Affected versions</td><td>All</td></tr>
  <tr><td>CVE-ID</td><td>TBD</td></tr>
  <tr><td>Weakness</td><td>CWE-115: Misinterpretation of Input</td></tr>
  <tr><td>CVSS rating</td><td>9.8 (CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H)</td></tr>
</table>

## Description

Go's `encoding/xml` handles namespace prefixes on XML elements in a way that causes crafted markup to mutate during round-trips through the `xml.Decoder` and `xml.Encoder` implementations. Encoding and decoding using Go's `encoding/xml` can change the observed namespace as well as the observed local name of a maliciously crafted XML element.

Affected applications include software that relies on XML integrity for security-sensitive decisions. Prominent examples of such applications include SAML and XML-DSig implementations.

## Impact

Mutations caused by encoding round-trips can lead to incorrect or conflicting decisions in affected applications. Equivalent lookups within an XML document can return different results during different stages of the document's lifecycle. Attempting to validate the structure of an XML document can succeed or fail depending on the number of encoding round-trips it has gone through.

As a specific example, an affected SAML implementation can interpret a SAML Assertion as signed, but then proceed to read values from an unsigned part of the same document due to namespace mutations between signature verification and data access. This can lead to full authentication bypass and arbitrary privilege escalation within the scope of a SAML Service Provider.

## Workaround

The `github.com/mattermost/xml-roundtrip-validator` module can detect unstable constructs in an XML document, including unstable element namespace prefixes. Invoking the validator on all untrusted markup and failing early if it returns an error can prevent these types of issue from being exploited in an otherwise affected application.
