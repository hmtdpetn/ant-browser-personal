package browser

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
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

var profileBadgeIconSizes = []int{16, 20, 24, 30, 32, 36, 40, 48, 60, 64, 72, 96, 128, 256}

func writeProfileBadgeICO(path string, label string, name string) error {
	type iconImage struct {
		size int
		data []byte
	}

	images := make([]iconImage, 0, len(profileBadgeIconSizes))
	for _, size := range profileBadgeIconSizes {
		var pngData bytes.Buffer
		if err := png.Encode(&pngData, newProfileBadgeImageAtSize(label, name, size)); err != nil {
			return err
		}
		images = append(images, iconImage{size: size, data: pngData.Bytes()})
	}

	var ico bytes.Buffer
	_ = binary.Write(&ico, binary.LittleEndian, uint16(0))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(len(images)))

	offset := uint32(6 + len(images)*16)
	for _, entry := range images {
		dimension := byte(entry.size)
		if entry.size == 256 {
			dimension = 0
		}
		ico.WriteByte(dimension)
		ico.WriteByte(dimension)
		ico.WriteByte(0)
		ico.WriteByte(0)
		_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
		_ = binary.Write(&ico, binary.LittleEndian, uint16(32))
		_ = binary.Write(&ico, binary.LittleEndian, uint32(len(entry.data)))
		_ = binary.Write(&ico, binary.LittleEndian, offset)
		offset += uint32(len(entry.data))
	}
	for _, entry := range images {
		_, _ = ico.Write(entry.data)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, ico.Bytes(), 0o644)
}

func newProfileBadgeImage(label string, name string) *image.RGBA {
	return newProfileBadgeImageAtSize(label, name, 128)
}

func newProfileBadgeImageAtSize(label string, name string, size int) *image.RGBA {
	if size < 1 {
		size = 1
	}
	const renderScale = 4
	canvas := renderProfileBadgeCanvas(label, name, size, renderScale)
	if canvas.Bounds().Dx() == size && canvas.Bounds().Dy() == size {
		return canvas
	}
	target := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(target, target.Bounds(), canvas, canvas.Bounds(), xdraw.Over, nil)
	return target
}

func renderProfileBadgeCanvas(label string, name string, targetSize int, scale int) *image.RGBA {
	labelRunes := []rune(strings.ToUpper(strings.TrimSpace(label)))
	if len(labelRunes) == 0 {
		labelRunes = []rune{'A'}
	}
	if len(labelRunes) > 2 {
		labelRunes = labelRunes[:2]
	}
	label = string(labelRunes)

	canvas := image.NewRGBA(image.Rect(0, 0, 128*scale, 128*scale))
	base, accent := badgeColors(label + "|" + name)
	shadow := color.RGBA{R: 3, G: 7, B: 18, A: 92}
	fillRoundedRect(canvas, scaledRect(8, 11, 120, 124, scale), 28*scale, shadow)

	outerTop := mixBadgeColor(accent, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 44)
	outerBottom := mixBadgeColor(base, color.RGBA{R: 2, G: 6, B: 20, A: 255}, 54)
	fillRoundedGradient(canvas, scaledRect(6, 5, 122, 121, scale), 30*scale, outerTop, outerBottom)

	innerTop := mixBadgeColor(accent, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 22)
	innerBottom := mixBadgeColor(base, accent, 42)
	fillRoundedGradient(canvas, scaledRect(9, 8, 119, 118, scale), 27*scale, innerTop, innerBottom)
	strokeRoundedRect(canvas, scaledRect(10, 9, 118, 117, scale), 26*scale, maxInt(scale, 2),
		color.RGBA{R: 255, G: 255, B: 255, A: 38})

	// At tiny taskbar sizes, a clean silhouette and bold horizontal label are clearer than decoration.
	if targetSize >= 32 {
		fillCircle(canvas, 102*scale, 29*scale, 25*scale, color.RGBA{R: 255, G: 255, B: 255, A: 18})
		fillCircle(canvas, 25*scale, 102*scale, 30*scale, color.RGBA{R: accent.R, G: accent.G, B: accent.B, A: 24})
		fillRoundedGradient(canvas, scaledRect(18, 15, 91, 19, scale), 2*scale,
			color.RGBA{R: 255, G: 255, B: 255, A: 54}, color.RGBA{R: 255, G: 255, B: 255, A: 0})
	}

	drawProfileBadgeLabel(canvas, label, scale)
	return canvas
}

var (
	profileBadgeTypefaceOnce sync.Once
	profileBadgeTypeface     *opentype.Font
	profileBadgeTypefaceErr  error
)

func getProfileBadgeTypeface() (*opentype.Font, error) {
	profileBadgeTypefaceOnce.Do(func() {
		profileBadgeTypeface, profileBadgeTypefaceErr = opentype.Parse(gobold.TTF)
	})
	return profileBadgeTypeface, profileBadgeTypefaceErr
}

func drawProfileBadgeLabel(dst *image.RGBA, label string, scale int) {
	typeface, err := getProfileBadgeTypeface()
	if err != nil {
		return
	}
	fontSize := 80.0
	if len([]rune(label)) > 1 {
		fontSize = 61.0
	}
	face, err := opentype.NewFace(typeface, &opentype.FaceOptions{
		Size:    fontSize * float64(scale),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return
	}

	drawer := &font.Drawer{Dst: dst, Face: face}
	advance := drawer.MeasureString(label)
	metrics := face.Metrics()
	centerX := fixed.I(64 * scale)
	centerY := fixed.I(65 * scale)
	baseline := centerY + (metrics.Ascent-metrics.Descent)/2
	start := centerX - advance/2

	// A compact outline preserves edge contrast at 16?24 px without returning to pixel glyphs.
	outline := image.NewUniform(color.RGBA{R: 2, G: 8, B: 24, A: 150})
	for _, offset := range []image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {2, 2}} {
		drawer.Src = outline
		drawer.Dot = fixed.Point26_6{
			X: start + fixed.I(offset.X*scale),
			Y: baseline + fixed.I(offset.Y*scale),
		}
		drawer.DrawString(label)
	}
	drawer.Src = image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	drawer.Dot = fixed.Point26_6{X: start, Y: baseline}
	drawer.DrawString(label)
}

func badgeColors(seed string) (color.RGBA, color.RGBA) {
	palette := []color.RGBA{
		{R: 59, G: 130, B: 246, A: 255},
		{R: 20, G: 184, B: 166, A: 255},
		{R: 249, G: 115, B: 22, A: 255},
		{R: 139, G: 92, B: 246, A: 255},
		{R: 236, G: 72, B: 153, A: 255},
		{R: 34, G: 197, B: 94, A: 255},
		{R: 6, G: 182, B: 212, A: 255},
		{R: 245, G: 158, B: 11, A: 255},
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	accent := palette[int(hash.Sum32())%len(palette)]
	base := mixBadgeColor(color.RGBA{R: 8, G: 15, B: 34, A: 255}, accent, 35)
	return base, accent
}

func scaledRect(x0 int, y0 int, x1 int, y1 int, scale int) image.Rectangle {
	return image.Rect(x0*scale, y0*scale, x1*scale, y1*scale)
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func fillCircle(img *image.RGBA, cx int, cy int, radius int, c color.RGBA) {
	fillRoundedRect(img, image.Rect(cx-radius, cy-radius, cx+radius, cy+radius), radius, c)
}

func fillRoundedRect(img *image.RGBA, rect image.Rectangle, radius int, c color.RGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if pointInRoundedRect(x, y, rect, radius) {
				blendBadgePixel(img, x, y, c)
			}
		}
	}
}

func fillRoundedGradient(img *image.RGBA, rect image.Rectangle, radius int, top color.RGBA, bottom color.RGBA) {
	height := rect.Dy() - 1
	if height < 1 {
		height = 1
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		mix := (y - rect.Min.Y) * 255 / height
		rowColor := mixBadgeColor(top, bottom, mix)
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if pointInRoundedRect(x, y, rect, radius) {
				blendBadgePixel(img, x, y, rowColor)
			}
		}
	}
}

func strokeRoundedRect(img *image.RGBA, rect image.Rectangle, radius int, width int, c color.RGBA) {
	if width < 1 {
		width = 1
	}
	inner := rect.Inset(width)
	innerRadius := radius - width
	if innerRadius < 0 {
		innerRadius = 0
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if pointInRoundedRect(x, y, rect, radius) && !pointInRoundedRect(x, y, inner, innerRadius) {
				blendBadgePixel(img, x, y, c)
			}
		}
	}
}

func pointInRoundedRect(x int, y int, rect image.Rectangle, radius int) bool {
	if !image.Pt(x, y).In(rect) {
		return false
	}
	if radius <= 0 {
		return true
	}
	maxRadius := rect.Dx() / 2
	if rect.Dy()/2 < maxRadius {
		maxRadius = rect.Dy() / 2
	}
	if radius > maxRadius {
		radius = maxRadius
	}
	if x >= rect.Min.X+radius && x < rect.Max.X-radius {
		return true
	}
	if y >= rect.Min.Y+radius && y < rect.Max.Y-radius {
		return true
	}
	cx := rect.Min.X + radius
	if x >= rect.Max.X-radius {
		cx = rect.Max.X - radius - 1
	}
	cy := rect.Min.Y + radius
	if y >= rect.Max.Y-radius {
		cy = rect.Max.Y - radius - 1
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= radius*radius
}

func mixBadgeColor(left color.RGBA, right color.RGBA, amount int) color.RGBA {
	if amount < 0 {
		amount = 0
	}
	if amount > 255 {
		amount = 255
	}
	inverse := 255 - amount
	return color.RGBA{
		R: uint8((int(left.R)*inverse + int(right.R)*amount) / 255),
		G: uint8((int(left.G)*inverse + int(right.G)*amount) / 255),
		B: uint8((int(left.B)*inverse + int(right.B)*amount) / 255),
		A: uint8((int(left.A)*inverse + int(right.A)*amount) / 255),
	}
}

func blendBadgePixel(img *image.RGBA, x int, y int, source color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) || source.A == 0 {
		return
	}
	destination := img.RGBAAt(x, y)
	sa := int(source.A)
	da := int(destination.A)
	outA := sa + da*(255-sa)/255
	if outA == 0 {
		return
	}
	remaining := da * (255 - sa) / 255
	img.SetRGBA(x, y, color.RGBA{
		R: uint8((int(source.R)*sa + int(destination.R)*remaining) / outA),
		G: uint8((int(source.G)*sa + int(destination.G)*remaining) / outA),
		B: uint8((int(source.B)*sa + int(destination.B)*remaining) / outA),
		A: uint8(outA),
	})
}
