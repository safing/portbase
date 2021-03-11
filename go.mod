module github.com/safing/portbase

go 1.15

require (
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/VictoriaMetrics/metrics v1.15.2
	github.com/aead/serpent v0.0.0-20160714141033-fba169763ea6
	github.com/armon/go-radix v1.0.0
	github.com/bluele/gcache v0.0.2
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger v1.6.2
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-version v1.2.1
	github.com/pkg/errors v0.9.1 // indirect
	github.com/seehuhn/fortuna v1.0.1
	github.com/shirou/gopsutil v3.21.2+incompatible
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.6.1
	github.com/tevino/abool v1.2.0
	github.com/tidwall/gjson v1.6.8
	github.com/tidwall/pretty v1.1.0 // indirect
	github.com/tidwall/sjson v1.1.5
	github.com/tklauser/go-sysconf v0.3.4 // indirect
	go.etcd.io/bbolt v1.3.5
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210309074719-68d13333faf2
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

// The follow-up commit removes Windows support.
// TODO: Check how we want to handle this in the future, possibly ingest
// needed functionality into here.
require github.com/google/renameio v0.1.1-0.20200217212219-353f81969824
