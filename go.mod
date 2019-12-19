module github.com/ystia/yorc/v4

// Makefile should also be updated when changing module major version (for injected variables)

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/SAP/go-hdb v0.14.1 // indirect
	github.com/SermoDigital/jose v0.9.1 // indirect
	github.com/abice/go-enum v0.1.4
	github.com/alecthomas/participle v0.3.0
	github.com/armon/go-metrics v0.3.0
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cheggaaa/pb/v3 v3.0.3
	github.com/containerd/continuity v0.0.0-20191214063359-1097c8bae83b // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20191128021309-1d7a30a10f73 // indirect
	github.com/docker/docker v0.0.0-20170504205632-89658bed64c2
	github.com/duosecurity/duo_api_golang v0.0.0-20190308151101-6c680f768e74 // indirect
	github.com/dustin/go-humanize v0.0.0-20171111073723-bb3d318650d4
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/gocql/gocql v0.0.0-20191126110522-1982a06ad6b9 // indirect
	github.com/google/addlicense v0.0.0-20190107131845-2e5cf00261bf
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/goware/urlx v0.3.1
	github.com/hashicorp/consul v1.2.3
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-memdb v1.0.4 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-plugin v1.0.0
	github.com/hashicorp/go-rootcerts v1.0.0
	github.com/hashicorp/vault v0.9.0
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jefferai/jsonx v1.0.1 // indirect
	github.com/julienschmidt/httprouter v1.2.0
	github.com/justinas/alice v0.0.0-20160512134231-052b8b6c18ed
	github.com/keybase/go-crypto v0.0.0-20190828182435-a05457805304 // indirect
	github.com/lib/pq v1.2.0 // indirect
	github.com/mgutz/logxi v0.0.0-20161027140823-aebf8a7d67ab // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/moby/moby v0.0.0-20170504205632-89658bed64c2
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/ory/dockertest v3.3.5+incompatible // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/satori/go.uuid v1.0.0
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/stevedomin/termtable v0.0.0-20150929082024-09d29f3fd628
	github.com/stretchr/testify v1.4.0
	github.com/tmc/dot v0.0.0-20180926222610-6d252d5ff882
	github.com/ystia/tdt2go v0.2.0 // indirect
	golang.org/x/crypto v0.0.0-20190325154230-a5d413f7728c
	golang.org/x/net v0.0.0-20190613194153-d28f0bde5980
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/AlecAivazis/survey.v1 v1.6.3
	gopkg.in/cookieo9/resources-go.v2 v2.0.0-20150225115733-d27c04069d0d
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/ory-am/dockertest.v3 v3.3.5 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
	vbom.ml/util v0.0.0-20160121211510-db5cfe13f5cc
)

// Due to this capital letter thing we have troubles and we have to replace it explicitly
replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.1

go 1.13
