module my-app

go 1.23

require github.com/ftery0/zenqo v0.0.0

// replace는 로컬 개발용 — 실제 배포 후에는 이 줄 삭제하고 go get github.com/ftery0/zenqo@latest
replace github.com/ftery0/zenqo => ../..
