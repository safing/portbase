module github.com/safing/portbase

go 1.15

require (
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/VictoriaMetrics/metrics v1.12.3
	github.com/aead/serpent v0.0.0-20160714141033-fba169763ea6
	github.com/armon/go-radix v1.0.0
	github.com/bluele/gcache v0.0.0-20190518031135-bc40bd653833
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger v1.6.1
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-version v1.2.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/seehuhn/fortuna v1.0.1
	github.com/shirou/gopsutil v2.20.4+incompatible
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/tevino/abool v1.0.0
	github.com/tidwall/gjson v1.6.0
	github.com/tidwall/sjson v1.1.1
	go.etcd.io/bbolt v1.3.4
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f
	golang.org/x/tools v0.0.0-20210115202250-e0d201561e39 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c // indirect
)

require (
	// The follow-up commit removes Windows support.
	// TOOD: Check how we want to handle this in the future, possibly ingest
	// needed functionality into here.
	github.com/google/renameio v0.1.1-0.20200217212219-353f81969824
)
