package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	var (
		port      int
		host      string
		apnsTopic string
		count     int
		message   string
		sound     string
		options   string
		token     string
		dryrun    bool
		verbose   bool
		jsonFile  string
	)

	flag.IntVar(&count, "count", 1, "send count")
	flag.IntVar(&port, "port", 8003, "gunfish port")
	flag.StringVar(&host, "host", "localhost", "gunfish host")
	flag.StringVar(&apnsTopic, "apns-topic", "", "apns topic")
	flag.StringVar(&message, "message", "test notification", "push notification message")
	flag.StringVar(&sound, "sound", "default", "push notification sound (default: 'default')")
	flag.StringVar(&options, "options", "", "options (key1=value1,key2=value2...)")
	flag.StringVar(&token, "token", "", "apns device token (required)")
	flag.BoolVar(&dryrun, "dryrun", false, "dryrun")
	flag.BoolVar(&verbose, "verbose", false, "dryrun")
	flag.StringVar(&jsonFile, "json-file", "", "json input file")

	flag.Parse()

	if verbose {
		log.Printf("host: %s, port: %d, send count: %d", host, port, count)
	}

	opts := map[string]string{}
	payloads := make([]map[string]interface{}, count)
	if jsonFile == "" {
		if options != "" {
			for _, opt := range strings.Split(options, ",") {
				kv := strings.Split(opt, "=")
				key, val := kv[0], kv[1]
				opts[key] = val
			}
		}

		for i := 0; i < count; i++ {
			payloads[i] = map[string]interface{}{}
			payloads[i] = buildPayload(token, message, sound, apnsTopic, opts)
		}
	}

	if dryrun {
		log.Println("[dryrun] checks request payload:")
		if jsonFile == "" {
			out, err := json.MarshalIndent(payloads, "", "    ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
		}

		out, err := ioutil.ReadFile(jsonFile)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	if verbose && len(payloads) > 0 {
		log.Printf("post data: %#v", payloads)
	}
	endpoint := fmt.Sprintf("http://%s:%d/push/apns", host, port)
	req, err := newRequest(endpoint, jsonFile, payloads)
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func buildPayload(token, message, sound, apnsTopic string, opts map[string]string) map[string]interface{} {
	payload := map[string]interface{}{
		"aps": map[string]string{
			"alert": message,
			"sound": sound,
		},
	}
	for k, v := range opts {
		payload[k] = v
	}

	return map[string]interface{}{
		"payload": payload,
		"token":   token,
		"header": map[string]interface{}{
			"apns-topic": apnsTopic,
		},
	}
}

func newRequest(endpoint, jsonFile string, payloads []map[string]interface{}) (*http.Request, error) {
	if jsonFile == "" {
		b := &bytes.Buffer{}
		err := json.NewEncoder(b).Encode(payloads)
		if err != nil {
			return nil, err
		}
		return http.NewRequest(http.MethodPost, endpoint, b)
	}

	b, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}
	return http.NewRequest(http.MethodPost, endpoint, b)
}
