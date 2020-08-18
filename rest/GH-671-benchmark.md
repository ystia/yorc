# Benchmark comparison of pre and post implementation of GH-671

## Pre-GH-671

```bash
Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -tags testing -bench ^(BenchmarkGetDeployment)$

goos: linux
goarch: amd64
pkg: github.com/ystia/yorc/v4/rest
BenchmarkGetDeployment/BenchmarkGetDeployment-1-4                    613           1773530 ns/op          249218 B/op       1180 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-10-4                   222           4913325 ns/op          709286 B/op       1991 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-100-4                   38          37407939 ns/op         5280587 B/op       9951 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-1000-4                   3         359853481 ns/op        51034448 B/op      89248 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-10000-4                  1        4153240261 ns/op        509219896 B/op    882108 allocs/op
PASS
ok      github.com/ystia/yorc/v4/rest   20.050s
```

As we can see response time increase linearly with the number of tasks in the system.
This is not what we expect.

## Post-GH-671

```bash
Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -tags testing -bench ^(BenchmarkGetDeployment)$

goos: linux
goarch: amd64
pkg: github.com/ystia/yorc/v4/rest
BenchmarkGetDeployment/BenchmarkGetDeployment-1-4                   1311            951950 ns/op          142578 B/op        948 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-10-4                  1300            980104 ns/op          142625 B/op        949 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-100-4                 1464            996585 ns/op          142424 B/op        948 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-1000-4                1353            898168 ns/op          142408 B/op        948 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-10000-4               1430            926285 ns/op          142407 B/op        948 allocs/op
PASS
ok      github.com/ystia/yorc/v4/rest   31.512s
```

With this fix response time stay constant whatever is the number of tasks in the system.

## Comparison

```bash
$ benchcmp rest/old.txt rest/new.txt
benchmark                                                 old ns/op      new ns/op     delta
BenchmarkGetDeployment/BenchmarkGetDeployment-1-4         1773530        951950        -46.32%
BenchmarkGetDeployment/BenchmarkGetDeployment-10-4        4913325        980104        -80.05%
BenchmarkGetDeployment/BenchmarkGetDeployment-100-4       37407939       996585        -97.34%
BenchmarkGetDeployment/BenchmarkGetDeployment-1000-4      359853481      898168        -99.75%
BenchmarkGetDeployment/BenchmarkGetDeployment-10000-4     4153240261     926285        -99.98%

benchmark                                                 old allocs     new allocs     delta
BenchmarkGetDeployment/BenchmarkGetDeployment-1-4         1180           948            -19.66%
BenchmarkGetDeployment/BenchmarkGetDeployment-10-4        1991           949            -52.34%
BenchmarkGetDeployment/BenchmarkGetDeployment-100-4       9951           948            -90.47%
BenchmarkGetDeployment/BenchmarkGetDeployment-1000-4      89248          948            -98.94%
BenchmarkGetDeployment/BenchmarkGetDeployment-10000-4     882108         948            -99.89%

benchmark                                                 old bytes     new bytes     delta
BenchmarkGetDeployment/BenchmarkGetDeployment-1-4         249218        142578        -42.79%
BenchmarkGetDeployment/BenchmarkGetDeployment-10-4        709286        142625        -79.89%
BenchmarkGetDeployment/BenchmarkGetDeployment-100-4       5280587       142424        -97.30%
BenchmarkGetDeployment/BenchmarkGetDeployment-1000-4      51034448      142408        -99.72%
BenchmarkGetDeployment/BenchmarkGetDeployment-10000-4     509219896     142407        -99.97%
```
