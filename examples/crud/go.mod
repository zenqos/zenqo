module my-app

go 1.23

require github.com/zenqos/zenqo v0.0.0

// Local development only — remove this line after publishing and run: go get github.com/zenqos/zenqo@latest
replace github.com/zenqos/zenqo => ../..
