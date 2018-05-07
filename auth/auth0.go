package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/zenoss/glog"
	"fmt"
	"github.com/control-center/serviced/utils"
	"net/http"
	"encoding/json"
	"errors"
	"crypto/rsa"
	"encoding/pem"
	"crypto/x509"
)

// TODO: determine whether we need TP import for this code from https://auth0.com/docs/quickstart/backend/golang/01-authorization

type jwtAuth0Claims struct {
	Issuer        string `json:"iss,omitempty"`
	IssuedAt      int64  `json:"iat,omitempty"`
	ExpiresAt     int64  `json:"exp,omitempty"`
	Audience      []string `json:"aud,omitempty"`
	Groups        []string `json:"https://zenoss.com/groups,omitempty"`
	Subject       string   `json:"sub, omitempty"`
}

func (t *jwtAuth0Claims) Valid() error {
	if t.Expired() {
		return ErrRestTokenExpired
	}
	return nil
}
/*
type RestToken interface {
	Valid() error
	Expired() bool
	AuthToken() string
	RestToken() string
	ValidateRequestHash(r *http.Request) bool
	HasAdminAccess() bool
}*/

type Auth0Token interface {
	HasAdminAccess() bool
}

type jwtAuth0RestToken struct {
	*jwtAuth0Claims
	authIdentity Identity
	restToken    string
}

func (t *jwtAuth0Claims) Expired() bool {
	now := jwt.TimeFunc().UTC().Unix()
	return now >= t.ExpiresAt
}

func (t *jwtAuth0Claims) AuthToken() string {
	// TODO: implement
	glog.Error("Function jwtAuth0Claims.AuthToken called - needs implementation.")
	return ""
}

func (t *jwtAuth0Claims) RestToken() string {
	// TODO: implement
	glog.Error("Function jwtAuth0Claims.RestToken called - needs implementation.")
	return ""
}


//func (t *jwtAuth0Claims) ValidateRequestHash(r *http.Request) bool {
//	// TODO: implement properly
//	hash := GetRequestHash(r)
//	return bytes.Equal(t.ReqHash, hash)
//	glog.Error("Function jwtAuth0Claims.ValidateRequestHash called - needs implementation. (returns true for now, which is INSECURE)")
//	return true
//}

func (t *jwtAuth0Claims) HasAdminAccess() bool {
	//glog.V(0).Info("Checking Admin Access...")
	if !utils.StringInSlice("All Zenoss Employees", t.Groups) {
		glog.Warning("Auth0 Admin access denied - 'All Zenoss Employess' not found in Groups.")
		return false
	}
	//glog.V(0).Info("... Admin access granted.")
	return true
}

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

/*
TODO: Look into caching this, so we're not making a web request every time an authentication
request comes in.
*/
func getPemCert(token *jwt.Token) ([]byte, error) {
	//glog.V(0).Info("getPemCert() entry")
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
	//glog.V(0).Info("getRSAPublicKey() entry")
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


/*
	See https://auth0.com/docs/api-auth/tutorials/verify-access-token for information on
	validating auth0 tokens. Per https://jwt.io/, the jwt-go library validates exp,
	but not iss or sub.
*/
func ParseAuth0Token(token string) (Auth0Token, error) {
	//glog.V(0).Info("ParseAuth0Token(): ", token)
	claims := &jwtAuth0Claims{}
	identity := &jwtIdentity{}
	parsed, err := jwt.ParseWithClaims(token, claims, func (token *jwt.Token) (interface{}, error) {
		//glog.V(0).Info("in callback inside jwt.ParseWithClaims()")
		// Validate the algorithm matches the key
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			glog.Warning("error getting RSA key from PEM: ", ErrInvalidSigningMethod)
			return nil, ErrInvalidSigningMethod
		}

		// extract public key from token
		key, err := getRSAPublicKey(token)
		if err != nil {
			glog.Warning("error getting RSA key from PEM: ", err)
			return nil, fmt.Errorf("error getting RSA key from PEM: %v\n", err)
		}
		//glog.V(0).Info("ParseAuth0Token(): returning key: ", fmt.Sprintf("%+v", key))
		return key, nil
	})
	if err != nil {
		if verr, ok := err.(*jwt.ValidationError); ok {
			glog.Warning("Validation error from jwt.ParseWIthClaims(): ", verr)
			if verr.Inner != nil && (verr.Inner == ErrIdentityTokenExpired || verr.Inner == ErrIdentityTokenBadSig) {
				glog.Warning("ParseAuth0Token: nope: ", verr.Inner)
				return nil, verr.Inner
			}
			if verr.Errors&jwt.ValidationErrorExpired != 0 || verr.Inner != nil && verr.Inner == ErrRestTokenExpired {
				glog.Warning("ParseAuth0Token: nope: ", ErrRestTokenExpired)
				return nil, ErrRestTokenExpired
			}
			if verr.Errors&(jwt.ValidationErrorSignatureInvalid|jwt.ValidationErrorUnverifiable) != 0 {
				glog.Warning("ParseAuth0Token: nope: ", ErrRestTokenBadSig)
				return nil, ErrRestTokenBadSig
			}
			if verr.Errors&(jwt.ValidationErrorMalformed) != 0 {
				glog.Warning("ParseAuth0Token: nope: ", ErrBadRestToken)
				return nil, ErrBadRestToken
			}
			if verr.Inner != nil {
				glog.Warning("ParseAuth0Token: nope: ", verr.Inner)
				return nil, verr.Inner
			}
			if verr != nil {
				glog.Warning("ParseAuth0Token: nope: ", verr)
				return nil, verr
			}
		}
		glog.Warning("ParseAuth0Token: nope: ", err)
		return nil, err
	}
	//glog.V(0).Info("ParseAuth0Token: so far, so good...")
	if claims, ok := parsed.Claims.(*jwtAuth0Claims); ok && parsed.Valid {
		restToken := &jwtAuth0RestToken{}
		restToken.jwtAuth0Claims = claims
		restToken.authIdentity = identity
		restToken.restToken = token
		//glog.V(0).Info("ParseAuth0Token: success!")
		return restToken, nil
	}
	glog.Warning("ParseAuth0Token: womp, womp!: ", ErrIdentityTokenInvalid)
	return nil, ErrIdentityTokenInvalid
}

