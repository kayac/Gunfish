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

	var (
		port      int
		host      string
		apnsTopic string
		count     int
		message   string
		sound     string
		options   string
		token     string
	)

	flag.IntVar(&count, "count", 1, "send count")
	flag.IntVar(&port, "port", 8003, "gunfish port")
	flag.StringVar(&host, "host", "localhost", "gunfish host")
	flag.StringVar(&apnsTopic, "apns-topic", "", "apns topic")
	flag.StringVar(&message, "message", "test notification", "push notification message")
	flag.StringVar(&sound, "sound", "default", "push notification sound (default: 'default')")
	flag.StringVar(&options, "options", "", "options (key1=value1,key2=value2...)")
	flag.StringVar(&token, "token", "", "apns device token (required)")

	flag.Parse()

	log.Printf("host: %s, port: %d, send count: %d", host, port, count)

	opts := map[string]string{}
	if options != "" {
		for _, opt := range strings.Split(options, ",") {
			kv := strings.Split(opt, "=")
			key, val := kv[0], kv[1]
			opts[key] = val
		}
	}

	payloads := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		payloads[i] = map[string]interface{}{}
		payloads[i] = buildPayload(token, message, sound, apnsTopic, opts)
	}

	b := &bytes.Buffer{}
	if err := json.NewEncoder(b).Encode(payloads); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println("post data:", b.String())

	endpoint := fmt.Sprintf("http://%s:%d/push/apns", host, port)
	req, err := http.NewRequest(http.MethodPost, endpoint, b)
	if err != nil {
		log.Fatal(err)
		return
	}
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("err: %s", err)
		return
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("err: %s", err)
		return
	}
	log.Println("resp:", string(out))
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
