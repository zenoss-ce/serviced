// © Zenoss, Inc. 2018, all rights reserved.
// Use is subject to terms as shown in the License.zenoss file.

package web

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/zenoss/glog"
	"io/ioutil"
	"net/http"
	"strings"
)

// TODO: determine whether we need TP import for this code from https://auth0.com/docs/quickstart/backend/golang/01-authorization

type JSONWebkeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type Jwks struct {
	Keys []JSONWebkeys `json:"keys"`
}

func getPemCert(token *jwt.Token) ([]byte, error) {
	glog.V(0).Info("getPemCert() entry")
	cert := ""
	resp, err := http.Get("https://zenoss-dev.auth0.com/.well-known/jwks.json")

	if err != nil {
		glog.Warning("error getting well-known jwks: ", err)
		return []byte(cert), err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		glog.Warning("error decoding PEM Certificate: ", err)
		return []byte(cert), err
	}

	x5c := jwks.Keys[0].X5c
	for k, v := range x5c {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + v + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		glog.Warning("Unable to find appropriate key.")
		err := errors.New("unable to find appropriate key")
		return []byte(cert), err
	}

	return []byte(cert), nil
}

// TODO: possible credit to https://stackoverflow.com/a/33088784/7154147
func getRSAPublicKey(token *jwt.Token) (*rsa.PublicKey, error) {
	glog.V(0).Info("getRSAPublicKey() entry")
	certBytes, err := getPemCert(token)
	if err != nil {
		glog.Warning("error getting Pem Cert from auth0: ", err)
		return nil, err
	}
	block, _ := pem.Decode(certBytes)
	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		glog.Warning("error parsing certificate: ", err)
		return nil, err
	}
	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	return rsaPublicKey, nil
}

func getAuth0Token(authcode string) (string, error) {
	glog.V(0).Info("getAuth0Token() entry")
	result := ""
	// Call back to auth0 to get token
	tokenURL := "https://zenoss-dev.auth0.com/oauth/token"
	clientsecret := "1l953QOzQPBWfTVNbyNzDpHzyuE4EWszdVdavjHKdblNVGv40GrdEixKwjwy0Wvc"
	clientid := "xQF6jCIx6ZynvlvzT8ZWWrbOswcgCwH9"
	redirecturl := "http://10.87.130.69/static/auth0login.html"
	payloadstr := "{\"grant_type\":\"authorization_code\"," +
		"\"client_id\": \"" + clientid + "\"," +
		"\"client_secret\": \"" + clientsecret + "\"," +
		"\"code\": \"" + authcode + "\"," +
		"\"redirect_uri\": \"" + redirecturl + "\"}"
	payload := strings.NewReader(payloadstr)

	glog.V(0).Info("payloadstr = ", payloadstr)
	glog.V(0).Info("making POST request to ", tokenURL)
	req, _ := http.NewRequest("POST", tokenURL, payload)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		glog.V(0).Info("Error from auth0 POST request: ", err)
		return "", err
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	result = string(body)
	glog.V(2).Info("response:", res)
	glog.V(2).Info("body:", string(body))
	return result, nil
}

type Auth0TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}
