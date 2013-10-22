// package auth - Digest Authentication
// http://play.golang.org/p/ABoHSHoTmu
package auth

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Authorization struct {
	Username, Password, Realm, NONCE, QOP, Opaque, Algorithm string
}

func GetAuthorization(username, password string, resp *http.Response) *Authorization {
	header := resp.Header.Get("www-authenticate")
	parts := strings.SplitN(header, " ", 2)
	parts = strings.Split(parts[1], ", ")
	opts := make(map[string]string)

	for _, part := range parts {
		vals := strings.SplitN(part, "=", 2)
		key := vals[0]
		val := strings.Trim(vals[1], "\",")
		opts[key] = val
	}

	auth := Authorization{
		username, password,
		opts["realm"], opts["nonce"], opts["qop"], opts["opaque"], opts["algorithm"],
	}
	return &auth
}

func SetDigestAuth(r *http.Request, username, password string, resp *http.Response, nc int) *http.Request {
	auth := GetAuthorization(username, password, resp)
	auth_str := GetAuthString(auth, r.URL, r.Method, nc)
	r.Header.Add("Authorization", auth_str)
	// TODO : Need to return multiple values i.e. error value
	return r
}

func GetAuthString(auth *Authorization, url *url.URL, method string, nc int) string {
	a1 := auth.Username + ":" + auth.Realm + ":" + auth.Password
	h := md5.New()
	io.WriteString(h, a1)
	ha1 := hex.EncodeToString(h.Sum(nil))

	h = md5.New()
	a2 := method + ":" + url.Path
	io.WriteString(h, a2)
	ha2 := hex.EncodeToString(h.Sum(nil))

	nc_str := fmt.Sprintf("%08x", nc)
	hnc := "MTM3MDgw"

	respdig := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, auth.NONCE, nc_str, hnc, auth.QOP, ha2)
	h = md5.New()
	io.WriteString(h, respdig)
	respdig = hex.EncodeToString(h.Sum(nil))

	base := "username=\"%s\", realm=\"%s\", nonce=\"%s\", uri=\"%s\", response=\"%s\""
	base = fmt.Sprintf(base, auth.Username, auth.Realm, auth.NONCE, url.Path, respdig)
	if auth.Opaque != "" {
		base += fmt.Sprintf(", opaque=\"%s\"", auth.Opaque)
	}
	if auth.QOP != "" {
		base += fmt.Sprintf(", qop=\"%s\", nc=%s, cnonce=\"%s\"", auth.QOP, nc_str, hnc)
	}
	if auth.Algorithm != "" {
		base += fmt.Sprintf(", algorithm=\"%s\"", auth.Algorithm)
	}
	return "Digest " + base
}
