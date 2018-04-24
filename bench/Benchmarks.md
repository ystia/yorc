Benchmarks
==========


New Deployment 
--------------

### Before improvements:
```
go test -bench=. -benchtime=5s
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4         	       1	15171243702 ns/op
PASS
ok  	github.com/ystia/yorc/bench	17.864s

```

### Improvement 1: Store definitions (imports) in parallel
 
```
go test -bench=. -benchtime=5s
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4         	       1	8989072224 ns/op
PASS
ok  	github.com/ystia/yorc/bench	12.266s
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
ok  	github.com/ystia/yorc/bench	9.257s
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

### Improvement 3: Change Transport MaxIdleConns / MaxIdleConnsPerHost values to PubMaxRoutines value

```
go test -bench=. -benchtime=300s
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4         	      50	6144313435 ns/op
PASS
ok  	github.com/ystia/yorc/bench	345.164s
```

#### Results from previous:

```
benchmark                                                old ns/op      new ns/op      delta
BenchmarkHttpApiNewDeployment/HttpApiNewDeployment-4     6438261471     6144313435     -4.57%
```
