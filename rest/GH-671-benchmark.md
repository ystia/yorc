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
