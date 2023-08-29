module example.com/main

go 1.19

replace example.com/keycloak => ./keycloak

replace example.com/eurekaclient => ./eurekaclient

replace example.com/httpdm => ./httpdm

replace example.com/utils => ./utils

replace example.com/logdm => ./logdm

require (
	example.com/eurekaclient v0.0.0-00010101000000-000000000000
	example.com/httpdm v0.0.0-00010101000000-000000000000
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.7
	github.com/swaggo/swag v1.8.10
)

require (
	example.com/keycloak v0.0.0-00010101000000-000000000000 // indirect
	example.com/logdm v0.0.0-00010101000000-000000000000 // indirect
	example.com/utils v0.0.0-00010101000000-000000000000 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/coreos/go-oidc/v3 v3.5.0 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.4 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/rabbitmq/amqp091-go v1.8.1 // indirect
	github.com/xuanbo/eureka-client v0.0.5 // indirect
	github.com/xuanbo/requests v0.0.1 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/net v0.6.0 // indirect
	golang.org/x/oauth2 v0.5.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
