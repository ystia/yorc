Benchmarks
==========


New Deployment 
--------------

### Before improvements:
```
go test -bench=. -benchtime=5s
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4         	       1	15171243702 ns/op
PASS
ok  	novaforge.bull.com/starlings-janus/janus/bench	17.864s

```

### Improvement 1: Store definitions (imports) in parallel
 
```
go test -bench=. -benchtime=5s
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4         	       1	8989072224 ns/op
PASS
ok  	novaforge.bull.com/starlings-janus/janus/bench	12.266s
```

#### Results:
```
benchmark                                                old ns/op       new ns/op      delta
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4     15171243702     8989072224     -40.75%
```

### Improvement 2: Store all data in parallel
 
```
go test -bench=. -benchtime=5s
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4         	       5	1030761390 ns/op
PASS
ok  	novaforge.bull.com/starlings-janus/janus/bench	9.257s
```


#### Results:

* From improvement 1:
```
benchmark                                                old ns/op      new ns/op      delta
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4     8989072224     1030761390     -88.53%
```

* From initial:
```
benchmark                                                old ns/op       new ns/op      delta
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4     15171243702     1030761390     -93.21%
```
