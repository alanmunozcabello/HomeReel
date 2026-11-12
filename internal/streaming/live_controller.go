package streaming

import (
	"fmt"
	"sync"
	"time"
)

// LiveController keeps the optional, shared manual state of the TV channel.
// Without an override, the original clock-based schedule is used.
type LiveController struct {
	mu       sync.Mutex
	manualID int
	offset   int64
	started  time.Time
	paused   bool
}

func (c *LiveController) Position(list []PlaylistItem) (int, int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.manualID == 0 {
		id, pos := CalculateLiveBroadcast(list)
		return id, pos, false
	}
	var duration int64
	for _, item := range list {
		if item.ID == c.manualID {
			duration = item.Duration
			break
		}
	}
	if duration <= 0 {
		c.manualID = 0
		id, pos := CalculateLiveBroadcast(list)
		return id, pos, false
	}
	pos := c.offset
	if !c.paused {
		pos = (pos + int64(time.Since(c.started).Seconds())) % duration
	}
	return c.manualID, pos, c.paused
}

func (c *LiveController) Control(list []PlaylistItem, action string, id int, seek int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	currentID, currentPos := c.currentLocked(list)
	switch action {
	case "auto":
		c.manualID, c.paused = 0, false
		return nil
	case "pause":
		c.manualID, c.offset, c.paused = currentID, currentPos, true
		return nil
	case "play":
		if currentID == 0 {
			return fmt.Errorf("no hay video disponible")
		}
		c.manualID, c.offset, c.started, c.paused = currentID, currentPos, time.Now(), false
		return nil
	case "select":
		if !contains(list, id) {
			return fmt.Errorf("video no disponible")
		}
		c.manualID, c.offset, c.started, c.paused = id, 0, time.Now(), false
		return nil
	case "seek":
		if currentID == 0 {
			return fmt.Errorf("no hay video disponible")
		}
		c.manualID, c.offset, c.started = currentID, max(0, seek), time.Now()
		return nil
	default:
		return fmt.Errorf("accion invalida")
	}
}
func (c *LiveController) currentLocked(list []PlaylistItem) (int, int64) {
	if c.manualID == 0 {
		return CalculateLiveBroadcast(list)
	}
	var d int64
	for _, x := range list {
		if x.ID == c.manualID {
			d = x.Duration
		}
	}
	if d <= 0 {
		return 0, 0
	}
	p := c.offset
	if !c.paused {
		p = (p + int64(time.Since(c.started).Seconds())) % d
	}
	return c.manualID, p
}
func contains(list []PlaylistItem, id int) bool {
	for _, x := range list {
		if x.ID == id && x.Duration > 0 {
			return true
		}
	}
	return false
}
