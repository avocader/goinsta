package goinsta

import (
	"encoding/json"
)

// Timeline is the object to represent the main feed on instagram, the first page that shows the latest feeds of my following contacts.
type Timeline struct {
	inst *Instagram
}

func newTimeline(inst *Instagram) *Timeline {
	time := &Timeline{
		inst: inst,
	}
	return time
}

// Get returns latest media from timeline.
//
// For pagination use FeedMedia.Next()
func (time *Timeline) Get() *FeedMedia {
	insta := time.inst
	media := &FeedMedia{}
	media.inst = insta
	media.endpoint = urlTimeline
	return media
}

// Reel returns slice of Reel
func (time *Timeline) Stories(reason string) (*Tray, error) {
	body := time.inst.prepareDataQuery(map[string]interface{}{
		"supported_capabilities_new": supportedCapabilities,
		"reason":                     reason,
	})

	res, err := time.inst.sendRequest(&reqOptions{
		Endpoint: urlStories,
		Query:    body,
		IsPost:   true,
	})
	if err == nil {
		tray := &Tray{}
		err = json.Unmarshal(res, tray)
		if err != nil {
			return nil, err
		}
		tray.set(time.inst, urlStories)
		return tray, nil
	}
	return nil, err
}
