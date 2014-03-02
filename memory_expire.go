// Copyright 2013 by sdm. All rights reserved.

package mcache

import "time"

// startTick start a goroutine to check expire checking
func (mc *mcache) startTick() {
	if mc == nil {
		return
	}

	interval := TickInterval
	if interval < _minTickInterval {
		interval = _minTickInterval
	}

	mc.tick = time.Tick(interval)
	for {
		select {
		case <-mc.tick:
			mc.recycle()
		case <-mc.stop:
			return
		}
	}
}

func (mc *mcache) recycle() {
	keys := mc.expKeys()
	mc.DeleteMulti(keys)
}

func (mc *mcache) expKeys() (keys []string) {
	mc.RLock()
	defer mc.RUnlock()

	for k, v := range mc.items {
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
