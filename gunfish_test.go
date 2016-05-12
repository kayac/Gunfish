package gunfish

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

func BenchmarkGunfish(b *testing.B) {
	myprof := "mybench.prof"
	f, err := os.Create(myprof)
	if err != nil {
		b.Fatal(err)
	}

	b.StopTimer()
	go func() {
		StartServer(config, Test)
	}()
	time.Sleep(time.Second * 1)

	oj := `{"token":"token-x","payload":{"aps":{"alert":{"body":"message","title":"bench test"},"sound":"default"},"suboption":"test"}}`
	jsons := bytes.NewBufferString("[")
	jsons.WriteString(oj)
	for i := 0; i < 2500; i++ {
		jsons.WriteString("," + oj)
	}
	jsons.WriteString("]")

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := do(jsons)
		if err != nil {
			b.Fatal(err)
		}
	}
	pprof.WriteHeapProfile(f)
	defer f.Close()
}

func do(jsons *bytes.Buffer) error {
	u, err := url.Parse(fmt.Sprintf("http://localhost:%d/push/apns", config.Provider.Port))
	if err != nil {
		return err
	}
	client := &http.Client{}
	nreq, err := http.NewRequest("POST", u.String(), jsons)
	nreq.Header.Set("Content-Type", ApplicationJSON)
	if err != nil {
		return err
	}

	resp, err := client.Do(nreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
