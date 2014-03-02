// Copyright 2013 by sdm. All rights reserved.

package mcache

import "time"

// startTick start a goroutine to check expire checking
func (self *mcache) startTick() {
	if self == nil {
		return
	}

	interval := TickInterval
	if interval < _minTickInterval {
		interval = _minTickInterval
	}

	self.tick = time.Tick(interval)
	for {
		select {
		case <-self.tick:
			self.recycle()
		case <-self.stop:
			return
		}
	}
}

func (self *mcache) recycle() {
	keys := self.expKeys()
	self.DeleteMulti(keys)
}

func (self *mcache) expKeys() (keys []string) {
	self.RLock()
	defer self.RUnlock()

	for k, v := range self.items {
		if v.expired() {
			if keys == nil {
				keys = make([]string, 0, 255)
			}
			keys = append(keys, k)
		}
	}

	return
}

// stopTick can stop goroutine of expire
func stopTick(self *MCache) {
	self.stop <- true
}
