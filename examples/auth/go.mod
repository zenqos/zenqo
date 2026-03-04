module my-app

go 1.23

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/zenqos/zenqo v0.0.0
)

// Local development only — remove this line after publishing and run: go get github.com/zenqos/zenqo@latest
replace github.com/zenqos/zenqo => ../..
