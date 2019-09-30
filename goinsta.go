package goinsta

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

// Instagram represent the main API handler
//
// Profiles: Represents instragram's user profile.
// Account:  Represents instagram's personal account.
// Search:   Represents instagram's search.
// Timeline: Represents instagram's timeline.
// Activity: Represents instagram's user activity.
// Inbox:    Represents instagram's messages.
// Location: Represents instagram's locations.
//
// See Scheme section in README.md for more information.
//
// We recommend to use Export and Import functions after first Login.
//
// Also you can use SetProxy and UnsetProxy to set and unset proxy.
// Golang also provides the option to set a proxy using HTTP_PROXY env var.
type Instagram struct {
	user string
	pass string
	// device id: android-1923fjnma8123
	dID string
	// uuid: 8493-1233-4312312-5123
	uuid string
	// rankToken
	rankToken string
	// token
	token string
	// phone id
	pid string
	// ads id
	adid string
	//client session id
	sessionId string
	//
	device          device
	userAgent       string
	LastLogin       int64
	LastExperiments int64

	LoggedIn bool

	// Instagram objects

	// Profiles is the user interaction
	Profiles *Profiles
	// Account stores all personal data of the user and his/her options.
	Account *Account
	// Search performs searching of multiple things (users, locations...)
	Search *Search
	// Timeline allows to receive timeline media.
	Timeline *Timeline
	// Activity are instagram notifications.
	Activity *Activity
	// Inbox are instagram message/chat system.
	Inbox *Inbox
	// Feed for search over feeds
	Feed *Feed
	// User contacts from mobile address book
	Contacts *Contacts
	// Location instance
	Locations *LocationInstance

	c *http.Client
}

// SetHTTPClient sets http client.  This further allows users to use this functionality
// for HTTP testing using a mocking HTTP client Transport, which avoids direct calls to
// the Instagram, instead of returning mocked responses.
func (inst *Instagram) SetHTTPClient(client *http.Client) {
	inst.c = client
}

// SetDeviceID sets device id
func (inst *Instagram) SetDeviceID(id string) {
	inst.dID = id
}

// SetUUID sets uuid
func (inst *Instagram) SetUUID(uuid string) {
	inst.uuid = uuid
}

// SetPhoneID sets phone id
func (inst *Instagram) SetPhoneID(id string) {
	inst.pid = id
}

// New creates Instagram structure
func New(username string, password string, device string, proxy *url.URL, session *Session) (*Instagram, error) {
	var inst *Instagram
	if session == nil {
		// this call never returns error
		jar, _ := cookiejar.New(nil)
		inst = &Instagram{
			user: username,
			pass: password,
			dID: generateDeviceID(
				generateMD5Hash(username + password),
			),
			adid:      generateUUID(),
			sessionId: generateUUID(),
			uuid:      generateUUID(), // both uuid must be differents
			pid:       generateUUID(),
			c: &http.Client{
				Transport: &http.Transport{
					Proxy: func(request *http.Request) (i *url.URL, e error) {
						return proxy, nil
					},
				},
				Jar: jar,
			},
			LoggedIn: false,
		}
		if v, contains := devices[device]; contains {
			inst.device = v
		} else {
			inst.device = devices[defaultDevice]
		}
		inst.userAgent = fmt.Sprintf("Instagram %s Android (%s/%s; %s; %s; %s; %s; %s; %s; ru_RU)",
			inst.device.InstagramVersion,
			inst.device.AndroidVersion,
			inst.device.AndroidRelease,
			inst.device.Dpi,
			inst.device.Resolution,
			inst.device.Manufacturer,
			inst.device.Device,
			inst.device.Model,
			inst.device.Cpu)
	} else {
		jar, _ := cookiejar.New(nil)
		urlApi, err := url.Parse(goInstaAPIUrl)
		if err != nil {
			return nil, err
		}
		jar.SetCookies(urlApi, mapToCookies(session.Cookies))
		inst = &Instagram{
			user:      username,
			pass:      password,
			dID:       session.UUIDs.DeviceID,
			adid:      session.UUIDs.AdvertisingID,
			sessionId: session.UUIDs.ClientSessionID,
			uuid:      session.UUIDs.UUID, // both uuid must be differents
			pid:       session.UUIDs.PhoneID,
			token:     session.UUIDs.Token,
			rankToken: session.UUIDs.RankToken,
			c: &http.Client{
				Transport: &http.Transport{
					Proxy: func(request *http.Request) (i *url.URL, e error) {
						return proxy, nil
					},
				},
				Jar: jar,
			},
			device:    session.Device,
			userAgent: session.UserAgent,
			LoggedIn:  true,
		}
	}
	inst.init()
	return inst, nil
}

func (inst *Instagram) init() {
	inst.Profiles = newProfiles(inst)
	inst.Activity = newActivity(inst)
	inst.Timeline = newTimeline(inst)
	inst.Search = newSearch(inst)
	inst.Inbox = newInbox(inst)
	inst.Feed = newFeed(inst)
	inst.Contacts = newContacts(inst)
	inst.Locations = newLocation(inst)
	inst.Account = &Account{inst: inst}
}

// SetProxy sets proxy for connection.
func (inst *Instagram) SetProxy(url1 string, insecure bool) error {
	uri, err := url.Parse(url1)
	if err == nil {
		inst.c.Transport = &http.Transport{
			Proxy: http.ProxyURL(uri),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		}
	}
	return err
}

// UnsetProxy unsets proxy for connection.
func (inst *Instagram) UnsetProxy() {
	inst.c.Transport = nil
}

// Export exports *Instagram object options
func (inst *Instagram) Export() (error, *Session) {
	urlApi, err := url.Parse(goInstaAPIUrl)
	if err != nil {
		return err, nil
	}

	return nil, &Session{
		//ID:          inst.Account.ID,
		User:        inst.user,
		Cookies:     cookiesToMap(inst.c.Jar.Cookies(urlApi)),
		Device:      inst.device,
		UUIDs:       UUIDs{PhoneID: inst.pid, UUID: inst.uuid, ClientSessionID: inst.sessionId, DeviceID: inst.dID, AdvertisingID: inst.adid, Token: inst.token, RankToken: inst.rankToken},
		UserAgent:   inst.userAgent,
		TimingValue: Timings{LastLogin: inst.LastLogin, LastExperiments: inst.LastExperiments},
	}
}

func cookiesToMap(cookies []*http.Cookie) map[string]string {
	res := make(map[string]string)
	for i, _ := range cookies {
		cookie := cookies[i]
		res[cookie.Name] = cookie.Value
	}
	return res
}

func mapToCookies(cookies map[string]string) []*http.Cookie {
	res := make([]*http.Cookie, 0)
	for k, v := range cookies {
		res = append(res, &http.Cookie{Name: k, Value: v})
	}
	return res
}

func (inst *Instagram) readMsisdnHeader() error {
	data, err := json.Marshal(
		map[string]string{
			"device_id":          inst.uuid,
			"mobile_subno_usage": "default",
		},
	)
	if err != nil {
		return err
	}
	_, err = inst.sendRequest(
		&reqOptions{
			Endpoint:   urlMsisdnHeader,
			IsPost:     true,
			Connection: "keep-alive",
			Query:      generateSignature(b2s(data)),
		},
	)
	return err
}

func (inst *Instagram) contactPrefill() error {
	data, err := json.Marshal(
		map[string]string{
			"phone_id":   inst.pid,
			"_csrftoken": inst.token,
			"usage":      "prefill",
		},
	)
	if err != nil {
		return err
	}
	_, err = inst.sendRequest(
		&reqOptions{
			Endpoint:   urlContactPrefill,
			IsPost:     true,
			Connection: "keep-alive",
			Query:      generateSignature(b2s(data)),
		},
	)
	return err
}

func (inst *Instagram) zrToken() error {
	_, err := inst.sendRequest(
		&reqOptions{
			Endpoint:   urlZrToken,
			IsPost:     false,
			Connection: "keep-alive",
			Query: map[string]string{
				"device_id":        inst.dID,
				"token_hash":       "",
				"custom_device_id": inst.uuid,
				"fetch_reason":     "token_expired",
			},
		},
	)
	return err
}

func (inst *Instagram) sendAdID() error {
	data, err := inst.prepareData(
		map[string]interface{}{
			"adid": inst.adid,
		},
	)
	if err != nil {
		return err
	}
	_, err = inst.sendRequest(
		&reqOptions{
			Endpoint:   urlLogAttribution,
			IsPost:     true,
			Connection: "keep-alive",
			Query:      generateSignature(data),
		},
	)
	return err
}

// Login performs instagram login.
//
// Password will be deleted after login
func (inst *Instagram) Login() error {
	if inst.LoggedIn {
		pullToRefresh := rand.Intn(100)%2 == 0
		var opts map[string]bool
		if pullToRefresh {
			opts = map[string]bool{"is_pull_to_refresh": false}
		} else {
			opts = map[string]bool{}
		}
		err := inst.Timeline.Get().Sync(opts)
		if err != nil {
			return err
		}
		if pullToRefresh {
			_, err = inst.Timeline.Stories("pull_to_refresh")
		} else {
			_, err = inst.Timeline.Stories("cold_start")
		}
		if err != nil {
			return err
		}
		isSessionExpired := (time.Now().Unix() - inst.LastLogin) > 1800
		if isSessionExpired {
			inst.sessionId = generateUUID()
			err = inst.Inbox.GetRankedRecipients("reshare", true)
			if err != nil {
				return err
			}
			err = inst.Inbox.GetRankedRecipients("save", true)
			if err != nil {
				return err
			}
			err = inst.Inbox.Sync()
			if err != nil {
				return err
			}
			err = inst.Inbox.GetPresence()
			if err != nil {
				return err
			}
			recent := inst.Activity.Recent()
			if !recent.Next() {
				return recent.err
			}
			err = inst.GetNotices()
			if err != nil {
				return err
			}
			err = inst.Explore(false)
			if err != nil {
				return err
			}
			inst.LastLogin = time.Now().Unix()
		}

		if (time.Now().Unix() - inst.LastExperiments) > 7200 {
			err = inst.syncUserFeatures()
			if err != nil {
				return err
			}
			err = inst.syncDeviceFeatures(false)
			if err != nil {
				return err
			}
			inst.LastExperiments = time.Now().Unix()
		}
		err = inst.Account.Sync()
		if err != nil {
			return err
		}
		return nil
	} else {

		err := inst.readMsisdnHeader()
		if err != nil {
			return err
		}

		err = inst.syncUserFeatures()
		if err != nil {
			return err
		}

		err = inst.syncDeviceFeatures(true)
		if err != nil {
			return err
		}

		err = inst.zrToken()
		if err != nil {
			return err
		}

		err = inst.sendAdID()
		if err != nil {
			return err
		}

		err = inst.contactPrefill()
		if err != nil {
			return err
		}

		result, err := json.Marshal(
			map[string]interface{}{
				"guid":                inst.uuid,
				"login_attempt_count": 0,
				"_csrftoken":          inst.token,
				"device_id":           inst.dID,
				"adid":                inst.adid,
				"phone_id":            inst.pid,
				"username":            inst.user,
				"password":            inst.pass,
				"google_tokens":       "[]",
			},
		)
		if err != nil {
			return err
		}
		body, err := inst.sendRequest(
			&reqOptions{
				Endpoint: urlLogin,
				Query:    generateSignature(b2s(result)),
				IsPost:   true,
				Login:    true,
			},
		)
		if err != nil {
			return err
		}
		inst.pass = ""

		// getting account data
		res := accountResp{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			return err
		}

		inst.Account = &res.Account
		inst.Account.inst = inst
		inst.rankToken = strconv.FormatInt(inst.Account.ID, 10) + "_" + inst.uuid
		inst.zrToken()
		return err
	}
}

// Logout closes current session
func (inst *Instagram) Logout() error {
	_, err := inst.sendSimpleRequest(urlLogout)
	inst.c.Jar = nil
	inst.c = nil
	return err
}

func (inst *Instagram) syncUserFeatures() error {
	data, err := inst.prepareData(
		map[string]interface{}{
			"id":                      inst.uuid,
			"server_config_retrieval": "1",
			"experiments":             launcherConfigs,
			"_uid":                    strconv.Itoa(int(inst.Account.ID)),
		},
	)
	if err != nil {
		return err
	}

	_, err = inst.sendRequest(
		&reqOptions{
			Endpoint: urlQeSync,
			Query:    generateSignature(data),
			IsPost:   true,
			Login:    true,
		},
	)
	return err
}

func (inst *Instagram) syncDeviceFeatures(login bool) error {
	data := map[string]interface{}{
		"id":                      inst.uuid,
		"server_config_retrieval": "1",
		"experiments":             loginExperiments,
	}
	if !login {
		data["_uuid"] = inst.uuid
		data["_uid"] = strconv.Itoa(int(inst.Account.ID))
		data["_csrftoken"] = inst.token
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = inst.sendRequest(
		&reqOptions{
			Endpoint: urlQeSync,
			Query:    generateSignature(b2s(body)),
			IsPost:   true,
			Login:    true,
		},
	)
	return err
}

func (inst *Instagram) megaphoneLog() error {
	data, err := inst.prepareData(
		map[string]interface{}{
			"id":        inst.Account.ID,
			"type":      "feed_aysf",
			"action":    "seen",
			"reason":    "",
			"device_id": inst.dID,
			"uuid":      generateMD5Hash(string(time.Now().Unix())),
		},
	)
	if err != nil {
		return err
	}
	_, err = inst.sendRequest(
		&reqOptions{
			Endpoint: urlMegaphoneLog,
			Query:    generateSignature(data),
			IsPost:   true,
			Login:    true,
		},
	)
	return err
}

func (inst *Instagram) expose() error {
	data, err := inst.prepareData(
		map[string]interface{}{
			"id":         inst.Account.ID,
			"experiment": "ig_android_profile_contextual_feed",
		},
	)
	if err != nil {
		return err
	}

	_, err = inst.sendRequest(
		&reqOptions{
			Endpoint: urlExpose,
			Query:    generateSignature(data),
			IsPost:   true,
		},
	)

	return err
}

// GetMedia returns media specified by id.
//
// The argument can be int64 or string
//
// See example: examples/media/like.go
func (inst *Instagram) GetMedia(o interface{}) (*FeedMedia, error) {
	media := &FeedMedia{
		inst:   inst,
		NextID: o,
	}
	return media, media.Sync(map[string]bool{})
}

func (insta *Instagram) Explore(isPrefetch bool) error {
	data := map[string]interface{}{
		"is_prefetch":                isPrefetch,
		"is_from_promote":            false,
		"timezone_offset":            "+0300",
		"session_id":                 insta.sessionId,
		"supported_capabilities_new": supportedCapabilities,
	}
	if isPrefetch {
		data["max_id"] = "0"
		data["module"] = "explore_popular"
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = insta.sendRequest(&reqOptions{
		Endpoint: "discover/explore/",
		Query:    generateSignature(b2s(body)),
		IsPost:   true,
	})
	return err
}

func (self *Instagram) GetNotices() error { //todo result
	_, err := self.sendSimpleRequest(urlUserNotice)
	return err

}
