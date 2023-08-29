package keycloak

import "fmt"
import logdm "example.com/logdm"
import oidc "github.com/coreos/go-oidc/v3/oidc"
import "github.com/golang-jwt/jwt"
import "golang.org/x/oauth2"
import "net/http"
import "strings"
import "context"

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
		logdm.WriteLogLine(method+" errot in mapping token. ")
		return ""
	}
}
