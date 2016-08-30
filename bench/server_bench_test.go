package bench

import (
	"archive/zip"
	"bytes"
	"github.com/hashicorp/consul/testutil"
	"io"
	"io/ioutil"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/server"
	"os"
	"path/filepath"
	"testing"
)

var response *http.Response

func zipDirContent(tb testing.TB, w *zip.Writer, rootEntry, dirPath string) error {
	//tb.Logf("Analyzing %q rootEntry %q", dirPath, rootEntry)
	fileInfos, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		fileName := rootEntry + fileInfo.Name()
		// Create a header based off of the fileinfo
		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return err
		}

		// If it's a file, set the compression method to deflate (leave directories uncompressed)
		if !fileInfo.IsDir() {
			header.Method = zip.Deflate
		}

		header.Name = fileName

		// Add a trailing slash if the entry is a directory
		if fileInfo.IsDir() {
			header.Name += "/"
		}

		// Get a writer in the archive based on our header
		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		filePath := filepath.Join(dirPath, fileInfo.Name())

		if !fileInfo.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(writer, file); err != nil {
				return err
			}
		} else {
			if err := zipDirContent(tb, w, fileName+"/", filePath); err != nil {
				return err
			}
		}

	}
	return nil
}

func zipCSAR(tb testing.TB, path string) ([]byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return []byte{}, err
	}
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	zipDirContent(tb, w, "", absPath)

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func setupServer(b *testing.B) (*testutil.TestServer, chan struct{}) {

	nw := noopWriter{}

	//log.SetDebug(true)
	//log.SetDebug(false)
	log.SetOutput(nw)
	srv1 := testutil.NewTestServerConfig(b, func(c *testutil.TestServerConfig) {
		c.LogLevel = "err"
		c.Stderr = nw
		c.Stdout = nw
	})

	configuration := config.Configuration{
		CONSUL_ADDRESS: srv1.HTTPAddr,
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
	if csarZip, err = zipCSAR(b, "../testdata/deployment/no-op"); err != nil {
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
	if err := os.RemoveAll("work"); err != nil {
		b.Fatal(err)
	}
}
