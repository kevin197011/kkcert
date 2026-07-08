package tz

import "time"

const Name = "Asia/Shanghai"

// Location is the system timezone (东八区).
var Location *time.Location

func init() {
	loc, err := time.LoadLocation(Name)
	if err != nil {
		Location = time.FixedZone("CST", 8*3600)
		return
	}
	Location = loc
}
