// fcmwebclient is a web app to receive FCM notifications in a browser,
// for testing end-to-end delivery via Gunfish.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed static
var static embed.FS

func main() {
	var (
		port           int
		firebaseConfig string
		vapidKey       string
		gunfishURL     string
	)
	flag.IntVar(&port, "port", 8080, "listen port (open http://localhost:<port>/ in a browser)")
	flag.StringVar(&firebaseConfig, "firebase-config", "", "path to Firebase web app config JSON (required)")
	flag.StringVar(&vapidKey, "vapid-key", "", "Web Push certificate key pair (VAPID public key) (required)")
	flag.StringVar(&gunfishURL, "gunfish", "http://localhost:8003", "Gunfish URL to send notifications")
	flag.Parse()

	if firebaseConfig == "" || vapidKey == "" {
		flag.Usage()
		os.Exit(1)
	}
	b, err := os.ReadFile(firebaseConfig)
	if err != nil {
		log.Fatal(err)
	}
	var conf map[string]any
	if err := json.Unmarshal(b, &conf); err != nil {
		log.Fatalf("invalid firebase config %s: %s", firebaseConfig, err)
	}
	for _, k := range []string{"apiKey", "projectId", "messagingSenderId", "appId"} {
		if conf[k] == nil {
			log.Fatalf("%s is not defined in %s", k, firebaseConfig)
		}
	}
	configJS, err := json.Marshal(map[string]any{"firebaseConfig": conf, "vapidKey": vapidKey})
	if err != nil {
		log.Fatal(err)
	}

	root, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(root))
	// config.js is loaded by both the page and the service worker.
	mux.HandleFunc("/config.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, "self.GUNFISH_FCM_TEST = %s;\n", configJS)
	})
	// Proxy to Gunfish, because Gunfish does not allow cross-origin requests.
	mux.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		res, err := http.Post(strings.TrimSuffix(gunfishURL, "/")+"/push/fcm/v1", "application/json", r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer res.Body.Close()
		w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
		w.WriteHeader(res.StatusCode)
		io.Copy(w, res.Body)
	})

	addr := fmt.Sprintf("localhost:%d", port)
	log.Printf("open http://%s/ in a browser (Gunfish: %s)", addr, gunfishURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}
