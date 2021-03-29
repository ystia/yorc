module github.com/ystia/yorc/v4

// Makefile should also be updated when changing module major version (for injected variables)

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/SAP/go-hdb v0.14.1 // indirect
	github.com/SermoDigital/jose v0.9.1 // indirect
	github.com/Sirupsen/logrus v0.1.0 // indirect
	github.com/abice/go-enum v0.2.3
	github.com/alecthomas/participle v0.3.0
	github.com/armon/go-metrics v0.3.0
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/cheggaaa/pb/v3 v3.0.3
	github.com/containerd/continuity v0.0.0-20200228182428-0f16d7a0959c // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20200206145737-bbfc9a55622e // indirect
	github.com/dgraph-io/ristretto v0.0.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.0.0-20170504205632-89658bed64c2
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/duosecurity/duo_api_golang v0.0.0-20200206192355-a9725220d6ca // indirect
	github.com/dustin/go-humanize v0.0.0-20171111073723-bb3d318650d4
	github.com/elastic/go-elasticsearch/v6 v6.8.6-0.20200428134631-c5be8f8ee116
	github.com/fatih/color v1.7.0
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/fvbommel/sortorder v1.0.1
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/gocql/gocql v0.0.0-20200228163523-cd4b606dd2fb // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/addlicense v0.0.0-20190107131845-2e5cf00261bf
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/goware/urlx v0.3.1
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/consul v1.2.3
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-memdb v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-plugin v1.0.0
	github.com/hashicorp/go-rootcerts v1.0.0
	github.com/hashicorp/serf v0.8.3 // indirect
	github.com/hashicorp/vault v0.9.0
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/jefferai/jsonx v1.0.1 // indirect
	github.com/julienschmidt/httprouter v1.2.0
	github.com/justinas/alice v0.0.0-20160512134231-052b8b6c18ed
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/matryer/resync v0.0.0-20161211202428-d39c09a11215
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mgutz/logxi v0.0.0-20161027140823-aebf8a7d67ab // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/moby/moby v0.0.0-20170504205632-89658bed64c2
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/ory/dockertest v3.3.5+incompatible // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.2.1
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/satori/go.uuid v1.0.0
	github.com/sethgrid/pester v0.0.0-20190127155807-68a33a018ad0 // indirect
	github.com/sethvargo/go-retry v0.1.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.6
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.4.0
	github.com/stevedomin/termtable v0.0.0-20150929082024-09d29f3fd628
	github.com/stretchr/testify v1.4.0
	github.com/tmc/dot v0.0.0-20180926222610-6d252d5ff882
	github.com/ystia/tdt2go v0.3.0
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/AlecAivazis/survey.v1 v1.6.3
	gopkg.in/cookieo9/resources-go.v2 v2.0.0-20150225115733-d27c04069d0d
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/ory-am/dockertest.v3 v3.3.5 // indirect
	gopkg.in/yaml.v2 v2.2.7
	gotest.tools v2.2.0+incompatible // indirect
	gotest.tools/v3 v3.0.0
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190709113604-33be087ad058 // indirect
)

// Due to this capital letter thing we have troubles and we have to replace it explicitly
replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.1

exclude github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000

go 1.13
