package goinsta

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type reqOptions struct {
	// Connection is connection header. Default is "close".
	Connection string

	// Login process
	Login bool

	// Endpoint is the request path of instagram api
	Endpoint string

	// IsPost set to true will send request with POST method.
	//
	// By default this option is false.
	IsPost bool

	// UseV2 is set when API endpoint uses v2 url.
	UseV2 bool

	// Query is the parameters of the request
	//
	// This parameters are independents of the request method (POST|GET)
	Query map[string]string
}

func (insta *Instagram) sendSimpleRequest(uri string, a ...interface{}) (body []byte, err error) {
	return insta.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(uri, a...),
		},
	)
}

func (insta *Instagram) sendRequest(o *reqOptions) (body []byte, err error) {
	method := "GET"
	if o.IsPost {
		method = "POST"
	}
	if o.Connection == "" {
		o.Connection = "keep-alive"
	}

	nu := goInstaAPIUrl
	if o.UseV2 {
		nu = goInstaAPIUrlv2
	}

	u, err := url.Parse(nu + o.Endpoint)
	if err != nil {
		return nil, err
	}

	vs := url.Values{}
	bf := bytes.NewBuffer([]byte{})

	for k, v := range o.Query {
		vs.Add(k, v)
	}

	if o.IsPost {
		bf.WriteString(vs.Encode())
	} else {
		for k, v := range u.Query() {
			vs.Add(k, strings.Join(v, " "))
		}

		u.RawQuery = vs.Encode()
	}

	var req *http.Request
	req, err = http.NewRequest(method, u.String(), bf)
	if err != nil {
		return
	}

	req.Header.Set("Connection", o.Connection)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", insta.userAgent)
	req.Header.Set("X-IG-App-ID", fbAnalytics)
	req.Header.Set("X-IG-Capabilities", igCapabilities)
	req.Header.Set("X-IG-Connection-Type", connType)
	req.Header.Set("X-IG-Connection-Speed", fmt.Sprintf("%dkbps", acquireRand(1000, 3700)))
	req.Header.Set("X-IG-Bandwidth-Speed-KBPS", strconv.FormatInt(acquireRand(5000, 20000), 10))
	req.Header.Set("X-IG-Bandwidth-TotalBytes-B", strconv.FormatInt(acquireRand(200000, 1000000), 10))
	req.Header.Set("X-IG-Bandwidth-TotalTime-MS", strconv.FormatInt(acquireRand(10, 170), 10))
	req.Header.Set("X-IG-Prefetch-Request", "foreground")
	req.Header.Set("X-IG-VP9-Capable", "false")
	req.Header.Set("X-FB-HTTP-Engine", "Liger")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip,deflate")
	req.Header.Set("Cookie2", "$Version=1")

	resp, err := insta.c.Do(req)
	if err != nil {
		return nil, err
	}

	u, _ = url.Parse(goInstaAPIUrl)
	for _, value := range insta.c.Jar.Cookies(u) {
		if strings.Contains(value.Name, "csrftoken") {
			insta.token = value.Value
		}
	}

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		defer reader.Close()
	default:
		reader = resp.Body
		defer resp.Body.Close()
	}

	body, err = ioutil.ReadAll(reader)
	if err == nil {
		err = isError(resp.StatusCode, body)
	}
	return body, err
}

func isError(code int, body []byte) (err error) {
	switch code {
	case 200:
	case 503:
		return Error503{
			Message: "Instagram API error. Try it later.",
		}
	case 400:
		ierr := Error400{}
		err = json.Unmarshal(body, &ierr)
		if err == nil && ierr.Payload.Message != "" {
			return ierr
		}
		fallthrough
	default:
		ierr := ErrorN{}
		err = json.Unmarshal(body, &ierr)
		if err != nil {
			return fmt.Errorf("Invalid status code: %d: %s", code, body)
		}
		if ierr.Message == "challenge_required" {
			ierr1 := ErrorChallenge{}
			err = json.Unmarshal(body, &ierr1)
			if err != nil {
				return fmt.Errorf("Invalid status code: %d: %s", code, body)
			}
			return ierr1
		}
		return ierr
	}
	return nil
}

func (insta *Instagram) prepareData(other ...map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"_uuid":      insta.uuid,
		"_csrftoken": insta.token,
	}
	/*if insta.Account != nil && insta.Account.ID != 0 {
		data["_uid"] = strconv.FormatInt(insta.Account.ID, 10)
	}*/
	if other != nil {
		for i := range other {
			for key, value := range other[i] {
				data[key] = value
			}
		}
	}
	b, err := json.Marshal(data)
	if err == nil {
		return b2s(b), err
	}
	return "", err
}

func (insta *Instagram) prepareDataQuery(other ...map[string]interface{}) map[string]string {
	data := map[string]string{
		"_uuid":      insta.uuid,
		"_csrftoken": insta.token,
	}
	for i := range other {
		for key, value := range other[i] {
			data[key] = toString(value)
		}
	}
	return data
}

func acquireRand(min, max int64) int64 {
	rand.Seed(time.Now().Unix())
	return rand.Int63n(max-min) + min
}
