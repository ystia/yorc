Benchmarks
==========


New Deployment 
--------------

### Before improvements:
Run 1:
```
go test -bench=. -benchtime=40s
testing: warning: no tests to run
BenchmarkHttpApiNewDeployment-4                      100         693002483 ns/op
BenchmarkParallelHttpApiNewDeployment-4              200         216811578 ns/op
PASS
ok      novaforge.bull.com/starlings-janus/janus/bench  158.768s
```
Run 2:
```
go test -bench=. -benchtime=40s
testing: warning: no tests to run
BenchmarkHttpApiNewDeployment-4                      100         714052804 ns/op
BenchmarkParallelHttpApiNewDeployment-4              200         237568800 ns/op
PASS
ok      novaforge.bull.com/starlings-janus/janus/bench  155.898s
```

