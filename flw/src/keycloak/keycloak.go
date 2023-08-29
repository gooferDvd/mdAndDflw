package keycloak

import "fmt"
import logdm "example.com/logdm"
import oidc "github.com/coreos/go-oidc/v3/oidc"
import "github.com/golang-jwt/jwt"
import "golang.org/x/oauth2"
import "net/http"
import "net/url"
import "strings"
import "context"
//import "strconv"
import "io/ioutil"
import "encoding/json"

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	RefreshExpiry int `json:"refresh_expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType   string `json:"token_type"`
	SessionState string `json:"session_state"`
	Scope       string `json:"scope"`
}

func Keycloak(ctx context.Context, clientId string, clientSecret string, oidcUrl string) *oidc.IDTokenVerifier {
    method:="Keycloak():"
	provider, err := oidc.NewProvider(ctx, oidcUrl)
	if err != nil {
		panic(err)
	}
	oauth2Config := oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),
		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID},
	}
	oidcConfig := &oidc.Config{
		ClientID: clientId,
	}
	logdm.WriteLogLine(method+" keycloak client id : " + oauth2Config.ClientID)
	return provider.Verifier(oidcConfig)

}

func TokenVerify(ctx context.Context, idTokenVerifier *oidc.IDTokenVerifier, w http.ResponseWriter, r *http.Request) bool {
    method:="TokenVerify():"
	rawAccessToken := r.Header.Get("Authorization")
	if rawAccessToken == "" {
		logdm.WriteLogLine(method+"token not found!")
		return false
	}
	parts := strings.Split(rawAccessToken, " ")
	if len(parts) != 2 {
		w.WriteHeader(400)
		return false
	}
	_, err := idTokenVerifier.Verify(ctx, parts[1])

	if err != nil {
		logdm.WriteLogLine(method+" error during verify the token.")
        logdm.WriteLogLine(method+" error:"+err.Error())
		return false
	}
	return true
}

func GetUserBytoken(r *http.Request) string {
    method := "GetUserBytoken():"
	rawAccessToken := r.Header.Get("Authorization")

	if rawAccessToken == "" {
		logdm.WriteLogLine(method+" token is missing!")
		return ""
	}
	parts := strings.Split(rawAccessToken, " ")
	if len(parts) != 2 {
		logdm.WriteLogLine(method+" token is invalid.")
		return ""
	}
	token, _, err := new(jwt.Parser).ParseUnverified(parts[1], jwt.MapClaims{})
	if err != nil {
		logdm.WriteLogLine(method+" error parsing the token.")
		return ""
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		name := fmt.Sprint(claims["preferred_username"])
		logdm.WriteLogLine(method+" received an operation on container from user:"+name)
		return name
	} else {
		logdm.WriteLogLine(method+" error in mapping token. ")
		return ""
	}
}

func GetToken(urlStr, clientID, clientSecret, username, password string) (*TokenResponse, error) {
	data := url.Values{}
	method := "GetToken() :"
	data.Set("grant_type", "password")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("username", username)
	data.Set("password", password)
    data.Set("scope", "openid")
	client := &http.Client{}
	r, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode())) 
	if err != nil {
		return nil, err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logdm.WriteLogLine (method + "error during read body request")
		return nil, err
	}
	tokenResp := &TokenResponse{}
	err = json.Unmarshal(body, tokenResp)
	if err != nil {
		logdm.WriteLogLine (method + "errore during read unmarshal token")
		
		return nil, err
	}
	return tokenResp, nil
}
