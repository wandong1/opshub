module github.com/ydcloud-dy/opshub/agent

go 1.25.0

require (
	github.com/creack/pty v1.1.24
	github.com/spf13/cobra v1.10.2
	github.com/ydcloud-dy/opshub v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.78.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/ydcloud-dy/opshub => ../
