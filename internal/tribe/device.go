package tribe

import (
	"strings"
)

// DeviceType is the kind of a detected device (EF-04); empty when unknown.
type DeviceType string

// Device types.
const (
	Phone    DeviceType = "phone"
	Tablet   DeviceType = "tablet"
	Computer DeviceType = "computer"
)

// Device is the detected device of a session (EF-04): type, operating system and browser
// from the User-Agent header, empty when unknown, and whether the session is in the
// installed app or a browser tab, as told by the client.
type Device struct {
	Type         DeviceType
	OS           string
	Browser      string
	InstalledApp bool
}

// DetectDevice deduces the device from the User-Agent header. User-Agent Client Hints
// wait for EF-04.
func DetectDevice(userAgent string, installedApp bool) Device {
	ua := userAgent
	has := func(token string) bool { return strings.Contains(ua, token) }

	var d Device
	d.InstalledApp = installedApp
	switch {
	case has("iPad"):
		d.Type, d.OS = Tablet, "iOS"
	case has("iPhone"), has("iPod"):
		d.Type, d.OS = Phone, "iOS"
	case has("Android"):
		d.OS = "Android"
		d.Type = Tablet
		if has("Mobile") {
			d.Type = Phone
		}
	case has("CrOS"):
		d.Type, d.OS = Computer, "ChromeOS"
	case has("Windows"):
		d.Type, d.OS = Computer, "Windows"
	case has("Macintosh"), has("Mac OS X"):
		d.Type, d.OS = Computer, "macOS"
	case has("Linux"):
		d.Type, d.OS = Computer, "Linux"
	}

	// Order matters: Edge, Opera and Samsung Internet also claim Chrome and Safari,
	// Chrome also claims Safari.
	switch {
	case has("Edg/"), has("EdgA/"), has("EdgiOS/"):
		d.Browser = "Edge"
	case has("SamsungBrowser/"):
		d.Browser = "Samsung Internet"
	case has("OPR/"), has("OPiOS/"):
		d.Browser = "Opera"
	case has("Firefox/"), has("FxiOS/"):
		d.Browser = "Firefox"
	case has("CriOS/"), has("Chrome/"), has("Chromium/"):
		d.Browser = "Chrome"
	case has("Safari/"):
		d.Browser = "Safari"
	case d.OS == "iOS" && has("AppleWebKit/"):
		// The installed app on iOS drops the Safari token.
		d.Browser = "Safari"
	}
	return d
}
