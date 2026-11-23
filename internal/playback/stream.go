package playback

import (
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
)

// CipherFields holds signature-related parameters for a format.
type CipherFields struct {
	URL       string
	Signature string
	SigKey    string
	N         string
}

// ExtractCipherFields reads url / signatureCipher from a format node.
func ExtractCipherFields(f gjson.Result) CipherFields {
	if u := f.Get("url").String(); u != "" {
		cf := CipherFields{URL: u, SigKey: "signature"}
		if parsed, err := url.Parse(u); err == nil {
			cf.N = parsed.Query().Get("n")
		}
		return cf
	}
	cipher := f.Get("signatureCipher").String()
	if cipher == "" {
		cipher = f.Get("cipher").String()
	}
	if cipher == "" {
		return CipherFields{}
	}
	vals, err := url.ParseQuery(cipher)
	if err != nil {
		return CipherFields{}
	}
	cf := CipherFields{
		URL:       vals.Get("url"),
		Signature: vals.Get("s"),
		SigKey:    vals.Get("sp"),
	}
	if cf.SigKey == "" {
		cf.SigKey = "signature"
	}
	if parsed, err := url.Parse(cf.URL); err == nil {
		cf.N = parsed.Query().Get("n")
	}
	if cf.N == "" {
		cf.N = vals.Get("n")
	}
	return cf
}

// ApplySignature attaches a deciphered signature to a stream URL.
func ApplySignature(rawURL, sig, sigKey string) string {
	if sig == "" {
		return rawURL
	}
	if sigKey == "" {
		sigKey = "signature"
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		return rawURL + sep + url.QueryEscape(sigKey) + "=" + url.QueryEscape(sig)
	}
	q := u.Query()
	q.Set(sigKey, sig)
	u.RawQuery = q.Encode()
	return u.String()
}
