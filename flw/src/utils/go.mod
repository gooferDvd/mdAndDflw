module example.com/utils

go 1.19

replace example.com/logdm => ../logdm

replace example.com/keycloak => ../keycloak

require (
	example.com/keycloak v0.0.0-00010101000000-000000000000
	example.com/logdm v0.0.0-00010101000000-000000000000
	github.com/rabbitmq/amqp091-go v1.8.1
)

require (
	github.com/coreos/go-oidc/v3 v3.5.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/net v0.6.0 // indirect
	golang.org/x/oauth2 v0.5.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
