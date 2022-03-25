module github.com/cappuccinotm/dastracker

go 1.18

replace github.com/cappuccinotm/dastracker/lib => ./lib

require (
	github.com/cappuccinotm/dastracker/lib v0.0.0-00010101000000-000000000000
	github.com/go-pkgz/repeater v1.1.3
	github.com/go-pkgz/syncs v1.1.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/logutils v1.0.0
	github.com/jessevdk/go-flags v1.5.0
	github.com/stretchr/testify v1.7.0
	go.etcd.io/bbolt v1.3.6
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)
