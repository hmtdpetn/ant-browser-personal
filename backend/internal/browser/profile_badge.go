package browser

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const ProfileBadgeIconName = "Ant Browser Badge.ico"

func (m *Manager) ProfileBadgeLabel(target *Profile) string {
	if target == nil {
		return "A"
	}
	profiles := make([]*Profile, 0, len(m.Profiles))
	for _, profile := range m.Profiles {
		if profile != nil {
			profiles = append(profiles, profile)
		}
	}
	return ProfileBadgeLabel(profiles, target.ProfileId)
}

func ProfileBadgeLabel(profiles []*Profile, targetID string) string {
	sortedProfiles := append([]*Profile(nil), profiles...)
	sort.SliceStable(sortedProfiles, func(i, j int) bool {
		leftCreated := strings.TrimSpace(sortedProfiles[i].CreatedAt)
		rightCreated := strings.TrimSpace(sortedProfiles[j].CreatedAt)
		if leftCreated != "" && rightCreated != "" && leftCreated != rightCreated {
			return leftCreated < rightCreated
		}
		return sortedProfiles[i].ProfileId < sortedProfiles[j].ProfileId
	})

	baseByID := make(map[string]string, len(sortedProfiles))
	countByBase := map[string]int{}
	for _, profile := range sortedProfiles {
		if profile == nil {
			continue
		}
		base := profileBadgeBase(profile.ProfileName)
		baseByID[profile.ProfileId] = base
		countByBase[base]++
	}

	seenByBase := map[string]int{}
	for _, profile := range sortedProfiles {
		if profile == nil {
			continue
		}
		base := baseByID[profile.ProfileId]
		seenByBase[base]++
		if profile.ProfileId == targetID {
			if countByBase[base] <= 1 {
				return base
			}
			return base + profileBadgeSuffix(seenByBase[base])
		}
	}

	return profileBadgeBase(targetID)
}

func profileBadgeBase(name string) string {
	name = strings.TrimSpace(name)
	for _, r := range name {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		if r >= 'a' && r <= 'z' {
			return string(r - 'a' + 'A')
		}
		if r >= 'A' && r <= 'Z' {
			return string(r)
		}
		if r >= '0' && r <= '9' {
			return string(r)
		}
		if letter, ok := chineseInitial(r); ok {
			return letter
		}
		if unicode.IsLetter(r) {
			upper := unicode.ToUpper(r)
			if upper >= 'A' && upper <= 'Z' {
				return string(upper)
			}
		}
	}
	return "A"
}

func profileBadgeSuffix(index int) string {
	const suffixes = "1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if index <= 0 {
		return "1"
	}
	if index <= len(suffixes) {
		return string(suffixes[index-1])
	}
	return string(suffixes[len(suffixes)-1])
}

func chineseInitial(r rune) (string, bool) {
	if r < '\u4e00' || r > '\u9fff' {
		return "", false
	}
	encoded, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte(string(r)))
	if err != nil || len(encoded) < 2 {
		return "Z", true
	}
	code := int(encoded[0]-0xA0)*100 + int(encoded[1]-0xA0)
	bounds := []struct {
		start  int
		letter string
	}{
		{1601, "A"}, {1637, "B"}, {1833, "C"}, {2078, "D"}, {2274, "E"},
		{2302, "F"}, {2433, "G"}, {2594, "H"}, {2787, "J"}, {3106, "K"},
		{3212, "L"}, {3472, "M"}, {3635, "N"}, {3722, "O"}, {3730, "P"},
		{3858, "Q"}, {4027, "R"}, {4086, "S"}, {4390, "T"}, {4558, "W"},
		{4684, "X"}, {4925, "Y"}, {5249, "Z"},
	}
	for i := len(bounds) - 1; i >= 0; i-- {
		if code >= bounds[i].start {
			return bounds[i].letter, true
		}
	}
	return "Z", true
}

func EnsureProfileBadgeIcon(userDataDir string, profileName string, label string) (string, error) {
	userDataDir = strings.TrimSpace(userDataDir)
	label = strings.ToUpper(strings.TrimSpace(label))
	if userDataDir == "" || label == "" {
		return "", nil
	}
	defaultDir := filepath.Join(userDataDir, "Default")
	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		return "", err
	}
	iconPath := filepath.Join(defaultDir, ProfileBadgeIconName)
	return iconPath, writeProfileBadgeICO(iconPath, label, profileName)
}

func writeProfileBadgeICO(path string, label string, name string) error {
	img := newProfileBadgeImage(label, name)
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		return err
	}

	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0, 128, 128, 0, 0, 1, 0, 32, 0})
	size := uint32(pngData.Len())
	offset := uint32(6 + 16)
	ico.Write([]byte{byte(size), byte(size >> 8), byte(size >> 16), byte(size >> 24)})
	ico.Write([]byte{byte(offset), byte(offset >> 8), byte(offset >> 16), byte(offset >> 24)})
	ico.Write(pngData.Bytes())

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, ico.Bytes(), 0o644)
}

func newProfileBadgeImage(label string, name string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	base, accent := badgeColors(label + "|" + name)
	draw.Draw(img, img.Bounds(), &image.Uniform{C: base}, image.Point{}, draw.Src)
	fillCircle(img, 64, 64, 56, accent)
	fillCircle(img, 64, 64, 47, color.RGBA{R: 255, G: 255, B: 255, A: 235})
	fillCircle(img, 64, 64, 41, accent)
	drawLabel(img, label)
	return img
}

func badgeColors(seed string) (color.RGBA, color.RGBA) {
	palette := []color.RGBA{
		{R: 32, G: 92, B: 190, A: 255},
		{R: 0, G: 132, B: 105, A: 255},
		{R: 184, G: 76, B: 31, A: 255},
		{R: 118, G: 72, B: 172, A: 255},
		{R: 184, G: 44, B: 83, A: 255},
		{R: 56, G: 118, B: 72, A: 255},
		{R: 34, G: 116, B: 158, A: 255},
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	accent := palette[int(hash.Sum32())%len(palette)]
	return color.RGBA{R: 20, G: 24, B: 32, A: 255}, accent
}

func fillCircle(img *image.RGBA, cx int, cy int, radius int, c color.RGBA) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func drawLabel(img *image.RGBA, label string) {
	if len(label) > 2 {
		label = label[:2]
	}
	scale := 9
	gap := 5
	if len(label) == 2 {
		scale = 6
		gap = 4
	}
	width := len(label)*5*scale + (len(label)-1)*gap
	startX := (128 - width) / 2
	startY := (128 - 7*scale) / 2
	for i, r := range label {
		drawGlyph(img, r, startX+i*(5*scale+gap), startY, scale, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
}

func drawGlyph(img *image.RGBA, r rune, x int, y int, scale int, c color.RGBA) {
	pattern, ok := badgeFont[r]
	if !ok {
		pattern = badgeFont['A']
	}
	for row, bits := range pattern {
		for col, bit := range bits {
			if bit != '1' {
				continue
			}
			rect := image.Rect(x+col*scale, y+row*scale, x+(col+1)*scale, y+(row+1)*scale)
			draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
		}
	}
}

var badgeFont = map[rune][]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01110", "10001", "10000", "10000", "10000", "10001", "01110"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01110", "10001", "10000", "10111", "10001", "10001", "01110"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"01110", "00100", "00100", "00100", "00100", "00100", "01110"},
	'J': {"00111", "00010", "00010", "00010", "10010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}
