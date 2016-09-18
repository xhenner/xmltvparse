package xmltvparse

import (
	"encoding/xml"
	"fmt"
	"time"
)

const xmldateformat = "20060102150400 -0700"

type xmltv struct {
	ChannelList   []xmlchannel   `xml:"channel"`
	ProgrammeList []xmlprogramme `xml:"programme"`
}

type xmlchannel struct {
	Id   string `xml:"id,attr"`
	Name string `xml:"display-name"`
}

type xmlprogramme struct {
	Start       string   `xml:"start,attr"`
	Stop        string   `xml:"stop,attr"`
	Channel     string   `xml:"channel,attr"`
	Title       string   `xml:"title"`
	SubTitle    string   `xml:"sub-title"`
	Description string   `xml:"desc"`
	Credits     string   `xml:"credits"`
	Date        string   `xml:"date"`
	Categories  []string `xml:"category"`
	Rating      string   `xml:"rating>value"`
}

type TvGrid map[time.Time][]Programme

type Programme struct {
	Start       time.Time `json:"start"        xml:"start,attr"`
	Stop        time.Time `json:"stop"         xml:"stop,attr"`
	Length      int       `json:"duration"     xml:"duration"`
	Channel     string    `json:"channel,attr" xml:"channel,attr"`
	Title       string    `json:"title"        xml:"title"`
	SubTitle    string    `json:"sub-title"    xml:"sub-title"`
	Description string    `json:"desc"         xml:"desc"`
	Credits     string    `json:"credits"      xml:"credits"`
	Date        string    `json:"date"         xml:"date"`
	Categories  []string  `json:"category"     xml:"category"`
	Rating      string    `json:"rating>"      xml:"rating>value"`
}

func (p Programme) String() string {
	return fmt.Sprintf("(%s) %s: %s - %dm", p.Channel, p.Start.Format("15:04"), p.Title, p.Length)
}

func (t *TvGrid) Load(s []byte) error {
	var x xmltv

	// get the structure
	// TODO: parse the file in 1 pass
	if err := xml.Unmarshal(s, &x); err != nil {
		return err
	}

	// there can be a list of all channel aliases
	channels := make(map[string]string)
	for _, v := range x.ChannelList {
		channels[v.Id] = v.Name
	}

	// (re)initialize the data structure
	newdata := make(TvGrid, len(x.ProgrammeList)/len(x.ChannelList))

	for _, v := range x.ProgrammeList {
		var entry Programme
		entry.Start, _ = time.Parse(xmldateformat, v.Start)
		entry.Stop, _ = time.Parse(xmldateformat, v.Stop)

		// use channel alias if possible
		if name, ok := channels[v.Channel]; ok {
			entry.Channel = name
		} else {
			entry.Channel = channels[v.Channel]
		}
		entry.Length = int(entry.Stop.Sub(entry.Start).Minutes())
		entry.Title = v.Title
		entry.SubTitle = v.SubTitle
		entry.Description = v.Description
		entry.Credits = v.Credits
		entry.Date = v.Date
		entry.Categories = v.Categories
		entry.Rating = v.Rating

		bucket := entry.Start.Round(time.Hour)
		if _, ok := newdata[bucket]; ok {
			newdata[bucket] = append(newdata[bucket], entry)
		} else {
			newdata[bucket] = []Programme{entry}
		}
	}

	*t = newdata
	return nil
}

func aroundTime(d time.Time) []time.Time {
	rnd := d.Round(time.Hour)
	return []time.Time{
		rnd.Add(-2 * time.Hour),
		rnd.Add(-1 * time.Hour),
		rnd,
		rnd.Add(time.Hour),
		rnd.Add(2 * time.Hour)}
}

// PlayingAround returns all shows around a given time. Used for printing a
// TV grid
func (t TvGrid) PlayingAround(d time.Time) map[string]*[]Programme {
	res := make(map[string]*[]Programme)
	for _, h := range aroundTime(d) {
		if _, ok := t[h]; !ok {
			continue
		}
		for _, prog := range t[h] {
			if _, ok := res[prog.Channel]; !ok {
				list := make([]Programme, 0)
				res[prog.Channel] = &list
			}
			*res[prog.Channel] = append(*(res[prog.Channel]), prog)
		}
	}
	return res
}

// PlayingAt returns current show on each channel, and the next one. Used for a
// simple widget
func (t TvGrid) PlayingAt(d time.Time) map[string]*[2]Programme {
	res := make(map[string]*[2]Programme)
	after := make(map[string]Programme)
	for _, h := range aroundTime(d) {
		if _, ok := t[h]; !ok {
			continue
		}
		for _, prog := range t[h] {
			if d.After(prog.Start) && d.Before(prog.Stop) {
				res[prog.Channel] = &[2]Programme{prog}
				continue
			}
			if d.Before(prog.Start) {
				if _, ok := after[prog.Channel]; !ok {
					after[prog.Channel] = prog
				} else {
					if prog.Start.Before(after[prog.Channel].Start) {
						after[prog.Channel] = prog
					}
				}
			}
		}
	}
	for k, v := range after {
		if _, ok := res[k]; ok {
			res[k][1] = v
		}
	}
	return res
}

// PlayingAroundNow is a simple alias to display a grid around the current time
func (t TvGrid) PlayingAroundNow() map[string]*[]Programme {
	return t.PlayingAround(time.Now())
}

// PlayingNow is a simple alias to display currently on air shows
func (t TvGrid) PlayingNow() map[string]*[2]Programme {
	return t.PlayingAt(time.Now())
}
