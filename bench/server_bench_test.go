package bench

import (
	"bytes"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/consul/testutil"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/ziputil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/server"
)

const defaultWorkingDirectory string = "work"

var response *http.Response

type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func setupServer(b *testing.B) (*testutil.TestServer, chan struct{}) {

	nw := noopWriter{}

	//log.SetDebug(true)
	//log.SetDebug(false)
	log.SetOutput(nw)
	srv1, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "err"
		c.Stderr = nw
		c.Stdout = nw
	})
	b.Fatalf("Failed to setup consul server: %v", err)

	configuration := config.Configuration{
		WorkingDirectory:     defaultWorkingDirectory,
		ConsulAddress:        srv1.HTTPAddr,
		ConsulPubMaxRoutines: config.DefaultConsulPubMaxRoutines,
	}
	shutdownCh := make(chan struct{})
	go func() {
		if err := server.RunServer(configuration, shutdownCh); err != nil {
			b.Fatalf("Can't run server: %v", err)
		}
	}()
	return srv1, shutdownCh
}

func BenchmarkHttpApiNewDeployment(b *testing.B) {
	srv1, shutdownCh := setupServer(b)
	defer srv1.Stop()
	var csarZip []byte
	var err error
	if csarZip, err = ziputil.ZipPath("../testdata/deployment/no-op"); err != nil {
		b.Fatal(err)
	}
	var r *http.Response
	b.ResetTimer()

	b.Run("HttpApiNewDeployment", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r, err = http.Post("http://localhost:8800/deployments", "application/zip", bytes.NewReader(csarZip))
			if err != nil {
				b.Fatalf("POST failed (iteration %d): %v", i, err)
			}
			if r.StatusCode != 201 {
				b.Fatalf("POST failed (iteration %d): Expecting HTTP Status code 201 got %d", i, r.StatusCode)
			}
		}
	})

	b.StopTimer()
	close(shutdownCh)
	if err := os.RemoveAll(defaultWorkingDirectory); err != nil {
		b.Fatal(err)
	}
}
