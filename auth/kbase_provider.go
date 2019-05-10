package auth

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	bserr "github.com/kbase/blobstore/errors"
)

//TODO CACHE token -> user & name -> valid
//TODO ERROR - go through errors and match to shock if necessary, and type so can be converted to http code

// KBaseProvider provides authentication based on the KBase auth server
// (https://github.com/kbase/auth2)
type KBaseProvider struct {
	url           url.URL
	adminRoles    *[]string
	endpointToken url.URL
	endpointMe    url.URL
}

// AdminRole is an option for NewKBaseProvider that designates that users with the specified
// KBase auth service role are blobstore admins.
func AdminRole(role string) func(*KBaseProvider) error {
	return func(kb *KBaseProvider) error {
		role = strings.TrimSpace(role)
		if role == "" {
			return bserr.WhiteSpaceError("role")
		}
		r := append(*kb.adminRoles, role)
		kb.adminRoles = &r
		return nil
	}
}

// NewKBaseProvider creates a new auth provider targeting the KBase auth server.
func NewKBaseProvider(url url.URL, options ...func(*KBaseProvider) error,
) (*KBaseProvider, error) {
	if !url.IsAbs() {
		return nil, errors.New("url must be absolute")
	}
	r := []string(nil)
	kb := &KBaseProvider{url: url, adminRoles: &r}
	for _, option := range options {
		err := option(kb)
		if err != nil {
			return nil, err
		}
	}
	token, _ := url.Parse("api/V2/token")
	kb.endpointToken = *token
	me, _ := url.Parse("api/V2/me")
	kb.endpointMe = *me
	// TODO LATER check url is valid when auth testmode root returns correct info
	// could also check custom roles are valid & clock skew, probably not worth it
	return kb, nil
}

// GetUser gets a user given a token.
func (kb *KBaseProvider) GetUser(token string) (*User, error) {
	if strings.TrimSpace(token) == "" {
		return nil, bserr.WhiteSpaceError("token")
	}
	tokenjson, err := get(kb.endpointToken, token)
	if err != nil {
		return nil, err
	}
	mejson, err := get(kb.endpointMe, token)
	if err != nil {
		return nil, err // not sure how to test this given the previous passed
	}
	roles := mejson["customroles"].([]interface{})
	isadmin := kb.isAdmin(&roles)
	//TODO CACHE return expiration time from token info
	return &User{UserName: tokenjson["user"].(string), IsAdmin: isadmin}, nil
}

// expects roles to be strings
func (kb *KBaseProvider) isAdmin(roles *[]interface{}) bool {
	if len(*roles) < 1 || len(*kb.adminRoles) < 1 {
		return false
	}
	rolemap := map[string]struct{}{}
	for _, r := range *kb.adminRoles {
		rolemap[r] = struct{}{}
	}
	for _, r := range *roles {
		delete(rolemap, r.(string))
	}
	return len(rolemap) < len(*kb.adminRoles)
}

func get(u url.URL, token string) (map[string]interface{}, error) {
	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	authenticate(&req.Header, token)
	req.Header.Add("accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err // dunno how to test this
	}
	return toJSON(res)
}

// modifies header in place
func authenticate(h *http.Header, token string) {
	h.Add("authorization", token)
}

// will close body
func toJSON(resp *http.Response) (map[string]interface{}, error) {
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(io.LimitReader(resp.Body, 10000))
	if err != nil {
		return nil, err // dunno how to test this easily
	}
	if _, err = resp.Body.Read(make([]byte, 1, 1)); err != io.EOF {
		// TODO LOG b
		return nil, errors.New("Unexpectedly long body from auth service")
	}
	var authresp map[string]interface{}
	err = json.Unmarshal(b, &authresp)
	if err != nil {
		// TODO LOG b.
		return nil, errors.New("Non-JSON response from KBase auth server, status code: " +
			strconv.Itoa(resp.StatusCode))
	}
	if resp.StatusCode > 399 { // should never see 100s or 300s
		// assume that we have a valid error response from the auth server at this point
		aerr := authresp["error"].(map[string]interface{})
		if aerr["apperror"] == "Invalid token" {
			return nil, errors.New("KBase auth server reported token was invalid")
		}
		// add more errors responses here
		// not sure how to easily test this
		return nil, errors.New("Error from KBase auth server: " + aerr["message"].(string))
	}
	return authresp, nil
}
