package tribe

import "testing"

func TestDetectDevice(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want Device
	}{
		{
			name: "iPhone, Safari",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			want: Device{Type: Phone, OS: "iOS", Browser: "Safari"},
		},
		{
			name: "iPhone, installed app",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
			want: Device{Type: Phone, OS: "iOS", Browser: "Safari"},
		},
		{
			name: "iPhone, Chrome",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/129.0.6668.69 Mobile/15E148 Safari/604.1",
			want: Device{Type: Phone, OS: "iOS", Browser: "Chrome"},
		},
		{
			name: "iPad, Firefox",
			ua:   "Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/131.0 Mobile/15E148 Safari/605.1.15",
			want: Device{Type: Tablet, OS: "iOS", Browser: "Firefox"},
		},
		{
			name: "Android phone, Chrome",
			ua:   "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
			want: Device{Type: Phone, OS: "Android", Browser: "Chrome"},
		},
		{
			name: "Android tablet, Samsung Internet",
			ua:   "Mozilla/5.0 (Linux; Android 13; SM-X700) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/26.0 Chrome/122.0.0.0 Safari/537.36",
			want: Device{Type: Tablet, OS: "Android", Browser: "Samsung Internet"},
		},
		{
			name: "Windows, Edge",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0",
			want: Device{Type: Computer, OS: "Windows", Browser: "Edge"},
		},
		{
			name: "macOS, Safari",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
			want: Device{Type: Computer, OS: "macOS", Browser: "Safari"},
		},
		{
			name: "Linux, Firefox",
			ua:   "Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0",
			want: Device{Type: Computer, OS: "Linux", Browser: "Firefox"},
		},
		{
			name: "ChromeOS, Chrome",
			ua:   "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
			want: Device{Type: Computer, OS: "ChromeOS", Browser: "Chrome"},
		},
		{
			name: "Windows, Opera",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 OPR/114.0.0.0",
			want: Device{Type: Computer, OS: "Windows", Browser: "Opera"},
		},
		{name: "unknown", ua: "curl/8.5.0", want: Device{}},
		{name: "empty", ua: "", want: Device{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectDevice(tc.ua, false); got != tc.want {
				t.Errorf("DetectDevice = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestDetectDeviceInstalledApp(t *testing.T) {
	if !DetectDevice("", true).InstalledApp || DetectDevice("", false).InstalledApp {
		t.Error("InstalledApp is not the value told by the client")
	}
}

func TestDeviceString(t *testing.T) {
	cases := []struct {
		device Device
		want   string
	}{
		{device: Device{Type: Phone, OS: "iOS", Browser: "Safari", InstalledApp: true}, want: "phone · iOS · Safari · installed app"},
		{device: Device{Type: Computer, OS: "Linux", Browser: "Firefox"}, want: "computer · Linux · Firefox · tab"},
		{device: Device{}, want: "tab"},
	}
	for _, tc := range cases {
		if got := tc.device.String(); got != tc.want {
			t.Errorf("%+v.String() = %q, want %q", tc.device, got, tc.want)
		}
	}
}
