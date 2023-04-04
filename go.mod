module github.com/ystia/yorc/v4

// Makefile should also be updated when changing module major version (for injected variables)

require (
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/abice/go-enum v0.2.3
	github.com/alecthomas/participle v0.3.0
	github.com/armon/go-metrics v0.3.10
	github.com/blang/semver v3.5.1+incompatible
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible // indirect
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/cheggaaa/pb/v3 v3.0.3
	github.com/containerd/containerd v1.6.0 // indirect
	github.com/dgraph-io/ristretto v0.0.1
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/docker/docker v20.10.24+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/elastic/go-elasticsearch/v6 v6.8.6-0.20200428134631-c5be8f8ee116
	github.com/fatih/color v1.9.0
	github.com/frankban/quicktest v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/fvbommel/sortorder v1.0.1
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/addlicense v0.0.0-20190107131845-2e5cf00261bf
	github.com/google/uuid v1.2.0
	github.com/goware/urlx v0.3.1
	github.com/hashicorp/consul/api v1.12.0
	github.com/hashicorp/consul/sdk v0.9.0
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-hclog v1.1.0
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-plugin v1.4.3
	github.com/hashicorp/go-retryablehttp v0.7.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/memberlist v0.3.1 // indirect
	github.com/hashicorp/serf v0.9.7 // indirect
	github.com/hashicorp/vault/api v1.1.0
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/julienschmidt/httprouter v1.3.0
	github.com/justinas/alice v0.0.0-20160512134231-052b8b6c18ed
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/matryer/resync v0.0.0-20161211202428-d39c09a11215
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.4.3
	github.com/moby/moby v20.10.12+incompatible
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/sethvargo/go-retry v0.1.0
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stevedomin/termtable v0.0.0-20150929082024-09d29f3fd628
	github.com/stretchr/testify v1.7.0
	github.com/tmc/dot v0.0.0-20180926222610-6d252d5ff882
	github.com/ystia/tdt2go v0.3.0
	golang.org/x/crypto v0.0.0-20220131195533-30dcbda58838
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220128215802-99c3d69c2c27 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.6.3
	gopkg.in/cookieo9/resources-go.v2 v2.0.0-20150225115733-d27c04069d0d
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.23.4
)

// Due to this capital letter thing we have troubles and we have to replace it explicitly
replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.1

exclude github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000

go 1.13
