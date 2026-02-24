module my-app

go 1.23

require github.com/zenqos/zenqo v0.0.0

// Local development — remove this line after publishing.
replace github.com/zenqos/zenqo => ../..
