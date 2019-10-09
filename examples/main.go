package main

import (
	"encoding/json"
	"fmt"
	"github.com/ryumaev/goinsta/v3"
	"io/ioutil"
	"os"
)

func main() {
	f, err := ioutil.ReadFile("fun.365-session.json")
	if err != nil {
		var session *goinsta.Session
		_ = json.Unmarshal(f, &session)
		api, err := goinsta.New("fun.365", "pabxme17Q", "oneplus_7", nil, nil)
		if err == nil {
			err = api.Login()
			if err != nil {
				println(fmt.Sprintf("%v", err))
			} else {
				user, err := api.Profiles.ByName("ruzal_yumaev")
				if err != nil {
					println(err)
				} else {
					println(user.Username)
					api.Inbox.New(user, "hello")
					err, session := api.Export()
					if err == nil {
						bytes, err := json.Marshal(session)
						if err == nil {
							f, err := os.Create(session.User + "-session.json")
							defer f.Close()
							if err == nil {
								f.Write(bytes)
							} else {
								println(fmt.Sprintf("%v", err))
							}
						} else {
							println(fmt.Sprintf("%v", err))
						}
					} else {
						println(fmt.Sprintf("%v", err))
					}
				}
			}
		} else {
			println(fmt.Sprintf("%v", err))
		}
	} else {
		println(fmt.Sprintf("%v", err))
	}
}
