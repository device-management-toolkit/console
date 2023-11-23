module github.com/jritsema/go-htmx-starter

go 1.20

require (
	github.com/jritsema/gotoolbox v0.8.0
	github.com/open-amt-cloud-toolkit/go-wsman-messages v1.8.4
	go.etcd.io/bbolt v1.3.8
)

require gopkg.in/validator.v2 v2.0.1 // indirect

replace github.com/open-amt-cloud-toolkit/go-wsman-messages => ../go-wsman-messages

require (
	golang.org/x/sys v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
)
