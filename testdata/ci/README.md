# Yorc CI BDD

This project contains Behavior Driven Development integration tests.
We use [GoDog](https://github.com/DATA-DOG/godog) as BDD framework.

## How to use this

### Installation prerequisites

First godog binary should be installed using `go get github.com/DATA-DOG/godog/cmd/godog`.

Then you need to have Yorc and Alien4Cloud installed. We recommend using [Yorc bootstrap](https://yorc.readthedocs.io/en/latest/bootstrap.html) to do it.

### Configuration

This project comes with several configuration options that could be given using environment variables or a config file.
We use [Viper](https://github.com/spf13/viper) to manage configuration so the config file format may be JSON, TOML, YAML, HCL, envfile or Java properties.
A dot `.` in configuration keys name represent a nested key separator.

Here are supported config options:

|        Config Key Name        |   Environment Variable Name   |      Default Value      |                               Description                               |
| ----------------------------- | ----------------------------- | ----------------------- | ----------------------------------------------------------------------- |
| keep_failed_applications      | KEEP_FAILED_APPLICATIONS      | `false`                 | Do not undeploy the application if test fails                           |
| alien4cloud.url               | ALIEN4CLOUD_URL               | `http://127.0.0.1:8088` | Alien4Cloud URL                                                         |
| alien4cloud.user              | ALIEN4CLOUD_USER              | `admin`                 | Alien4Cloud user name                                                   |
| alien4cloud.password          | ALIEN4CLOUD_PASSWORD          | `admin`                 | Alien4Cloud user password                                               |
| alien4cloud.ca                | ALIEN4CLOUD_CA                | no default              | Path or content of a SSL CA used to connect to a secured Alien4Cloud    |
| alien4cloud.orchestrator_name | ALIEN4CLOUD_ORCHESTRATOR_NAME | `yorc`                  | Name of the configured orchestrator in A4C to use to interact with Yorc |
| chromedp.no_headless          | CHROMEDP_NO_HEADLESS          | `false`                 | Specify if chromedp should run in headless mode                         |
| chromedp.timeout              | CHROMEDP_TIMEOUT              | `1m`                    | Timeout of chromedp execution should be expressed in Go duration format |

### Running

To run all tests without distinction simply run godog in the current folder:

```bash
ALIEN4CLOUD_URL="https://127.0.0.1:8088" godog
```

You can also specify a feature file:

```bash
godog features/TestDev.feature
```

For a better control you can also use tags:

```bash
godog -t "@premium && ~@slow,~@Monitoring"
```

This will run scenarios that are tagged `@premium` but not `~@slow` or `~@Monitoring`.
