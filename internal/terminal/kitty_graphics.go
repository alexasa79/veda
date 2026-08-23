package terminal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"sort"
)

type KittyImagePlacement struct {
	Key     string
	Path    string
	X       int
	Y       int
	Columns int
	Rows    int
}

func (t *Terminal) SetKittyImagePlacements(placements []KittyImagePlacement) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pendingKittyImages = append(t.pendingKittyImages[:0], placements...)
}

func (t *Terminal) updateKittyImages() string {
	if t.visibleKittyImages == nil {
		t.visibleKittyImages = make(map[string]KittyImagePlacement)
		t.kittyImageIDs = make(map[string]uint32)
		t.kittyUsedImageIDs = make(map[uint32]string)
		t.kittyTransmitted = make(map[string]bool)
	}

	current := make(map[string]KittyImagePlacement, len(t.pendingKittyImages))
	for _, placement := range t.pendingKittyImages {
		if placement.Key != "" && placement.Path != "" && placement.Columns > 0 && placement.Rows > 0 {
			current[placement.Key] = placement
		}
	}

	previousPathCount := kittyImagePathCounts(t.visibleKittyImages)
	currentPathCount := kittyImagePathCounts(current)
	var output bytes.Buffer

	removedPaths := make([]string, 0)
	for path := range previousPathCount {
		if currentPathCount[path] == 0 {
			removedPaths = append(removedPaths, path)
		}
	}
	sort.Strings(removedPaths)
	for _, path := range removedPaths {
		fmt.Fprintf(&output, "\x1b_Ga=d,d=I,i=%d,q=2;\x1b\\", t.imageIDForPath(path))
		delete(t.kittyTransmitted, path)
	}

	removedKeys := make([]string, 0)
	for key, previous := range t.visibleKittyImages {
		next, remains := current[key]
		if currentPathCount[previous.Path] > 0 && (!remains || next.Path != previous.Path) {
			removedKeys = append(removedKeys, key)
		}
	}
	sort.Strings(removedKeys)
	for _, key := range removedKeys {
		previous := t.visibleKittyImages[key]
		fmt.Fprintf(&output, "\x1b_Ga=d,d=i,i=%d,p=%d,q=2;\x1b\\",
			t.imageIDForPath(previous.Path), kittyPlacementID(key))
	}

	keys := make([]string, 0, len(current))
	for key := range current {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		placement := current[key]
		imageID := t.imageIDForPath(placement.Path)
		if !t.kittyTransmitted[placement.Path] {
			encodedPath := base64.StdEncoding.EncodeToString([]byte(placement.Path))
			fmt.Fprintf(&output, "\x1b_Ga=t,f=100,t=f,i=%d,q=2;%s\x1b\\", imageID, encodedPath)
			t.kittyTransmitted[placement.Path] = true
		}

		if previous, unchanged := t.visibleKittyImages[key]; unchanged && previous == placement {
			continue
		}
		fmt.Fprintf(&output, "\x1b[%d;%dH", placement.Y+1, placement.X+1)
		fmt.Fprintf(&output, "\x1b_Ga=p,i=%d,p=%d,c=%d,r=%d,C=1,z=-1,q=2;\x1b\\",
			imageID, kittyPlacementID(key), placement.Columns, placement.Rows)
	}

	t.visibleKittyImages = current
	return output.String()
}

func kittyImagePathCounts(placements map[string]KittyImagePlacement) map[string]int {
	counts := make(map[string]int)
	for _, placement := range placements {
		counts[placement.Path]++
	}
	return counts
}

func (t *Terminal) imageIDForPath(path string) uint32 {
	if id := t.kittyImageIDs[path]; id != 0 {
		return id
	}
	id := kittyHash("image\x00" + path)
	for id == 0 || (t.kittyUsedImageIDs[id] != "" && t.kittyUsedImageIDs[id] != path) {
		id++
	}
	t.kittyImageIDs[path] = id
	t.kittyUsedImageIDs[id] = path
	return id
}

func kittyPlacementID(key string) uint32 {
	id := kittyHash("placement\x00" + key)
	if id == 0 {
		return 1
	}
	return id
}

func kittyHash(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}
