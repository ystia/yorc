# Benchmark comparison of pre and post implementation of GH-671

## Pre-GH-671

```bash
Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -tags testing -bench ^(BenchmarkGetDeployment)$

goos: linux
goarch: amd64
pkg: github.com/ystia/yorc/v4/rest
BenchmarkGetDeployment/BenchmarkGetDeployment-1-4                522     2156783 ns/op    249247 B/op      1180 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-10-4               172     6509103 ns/op    708585 B/op      1989 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-100-4               30    36641276 ns/op   5283954 B/op      9957 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-1000-4               3   376045837 ns/op  51057704 B/op     90320 allocs/op
BenchmarkGetDeployment/BenchmarkGetDeployment-10000-4              1  3911817514 ns/op  509427128 B/op    893058 allocs/op
PASS
ok    github.com/ystia/yorc/v4/rest  221.152s
```
