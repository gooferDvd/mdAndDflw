module example.com/keycloak

go 1.19

replace example.com/logdm => ../logdm/

require (
	example.com/logdm v0.0.0-00010101000000-000000000000
	github.com/coreos/go-oidc/v3 v3.5.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	golang.org/x/oauth2 v0.5.0
)

require (
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/net v0.6.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
