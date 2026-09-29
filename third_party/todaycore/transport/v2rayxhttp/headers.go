// This file is derived from Xray-core (common/utils/browser.go), licensed
// under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.
//
// Xray masquerades XHTTP requests as browser fetch() calls. The header set and
// its ORDER-INDEPENDENT contents are observable by a censor, so this is a
// faithful port.
//
// Deliberate deviation: Xray seeds its PRNG with klauspost/cpuid data
// (CPU family/model/cores/cacheline). sing-box does not depend on cpuid, so we
// seed with an equivalent host-stable fingerprint. The effect is identical:
// the generated Chrome version is stable for a given machine but differs
// between machines. The produced version may differ by a few minor steps from
// what Xray on the same host would pick; this is not detectable on the wire
// because real Chrome populations span many versions.

package v2rayxhttp

import (
	"hash/fnv"
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func getRandomizer() *rand.Rand {
	fnvHash := fnv.New64()
	fnvHash.Write([]byte(runtime.GOOS + runtime.GOARCH + strconv.Itoa(runtime.NumCPU())))
	return rand.New(rand.NewSource(int64(fnvHash.Sum64())))
}

var globalRng = getRandomizer()

// chromeVersion suffers from deviation of a normal distribution, as in Xray.
func chromeVersion() int {
	// Start from Chrome 144, released on 2026.1.13.
	startVersion := 144
	timeStart := time.Date(2026, 1, 13, 0, 0, 0, 0, time.UTC).Unix() / 86400
	timeCurrent := time.Now().Unix() / 86400
	timeDiff := int(timeCurrent-timeStart-35) - int(math.Floor(math.Pow(globalRng.Float64(), 2)*105))
	return startVersion + (timeDiff / 35)
}

func firefoxVersion() int {
	timeCurrent := time.Now().Unix() / 86400
	timeStart := time.Date(2024, 7, 29, 0, 0, 0, 0, time.UTC).Unix() / 86400
	timeDiff := timeCurrent - timeStart - 25 - int64(math.Floor(math.Pow(globalRng.Float64(), 2)*50))
	return int(timeDiff/30) + 128
}

func curlVersion() string {
	timeCurrent := time.Now().Unix() / 86400
	timeStart := time.Date(2023, 3, 20, 0, 0, 0, 0, time.UTC).Unix() / 86400
	timeDiff := int(timeCurrent - timeStart - 60) - int(math.Floor(math.Pow(globalRng.Float64(), 2)*165))
	minorValue := timeDiff / 57
	return "8." + strconv.Itoa(minorValue) + ".0"
}

var safariMinorMap = [25]int{
	0, 0, 0, 1, 1,
	1, 2, 2, 2, 2, 3, 3, 3, 4, 4,
	4, 5, 5, 5, 5, 5, 6, 6, 6, 6,
}

func safariVersion() string {
	anchoredTime := time.Now()
	releaseYear := anchoredTime.Year()
	splitPoint := time.Date(releaseYear, 9, 23, 0, 0, 0, 0, time.UTC)
	delayedDays := int(math.Floor(math.Pow(globalRng.Float64(), 3) * 75))
	splitPoint = splitPoint.AddDate(0, 0, delayedDays)
	if anchoredTime.Compare(splitPoint) < 0 {
		releaseYear--
		splitPoint = time.Date(releaseYear, 9, 23, 0, 0, 0, 0, time.UTC)
		splitPoint = splitPoint.AddDate(0, 0, delayedDays)
	}
	minorVersion := safariMinorMap[(anchoredTime.Unix()-splitPoint.Unix())/1296000]
	return strconv.Itoa(releaseYear-1999) + "." + strconv.Itoa(minorVersion)
}

// Chromium brand GREASE implementation.
var (
	clientHintGreaseNA  = []string{" ", "(", ":", "-", ".", "/", ")", ";", "=", "?", "_"}
	clientHintVersionNA = []string{"8", "99", "24"}
	clientHintShuffle3  = [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	clientHintShuffle4  = [][4]int{
		{0, 1, 2, 3}, {0, 1, 3, 2}, {0, 2, 1, 3}, {0, 2, 3, 1}, {0, 3, 1, 2}, {0, 3, 2, 1},
		{1, 0, 2, 3}, {1, 0, 3, 2}, {1, 2, 0, 3}, {1, 2, 3, 0}, {1, 3, 0, 2}, {1, 3, 2, 0},
		{2, 0, 1, 3}, {2, 0, 3, 1}, {2, 1, 0, 3}, {2, 1, 3, 0}, {2, 3, 0, 1}, {2, 3, 1, 0},
		{3, 0, 1, 2}, {3, 0, 2, 1}, {3, 1, 0, 2}, {3, 1, 2, 0}, {3, 2, 0, 1}, {3, 2, 1, 0},
	}
)

func getGreasedChInvalidBrand(seed int) string {
	return "\"Not" + clientHintGreaseNA[seed%len(clientHintGreaseNA)] + "A" +
		clientHintGreaseNA[(seed+1)%len(clientHintGreaseNA)] + "Brand\";v=\"" +
		clientHintVersionNA[seed%len(clientHintVersionNA)] + "\""
}

func getGreasedChOrder(brandLength int, seed int) []int {
	switch brandLength {
	case 1:
		return []int{0}
	case 2:
		return []int{seed % brandLength, (seed + 1) % brandLength}
	case 3:
		return clientHintShuffle3[seed%len(clientHintShuffle3)][:]
	default:
		return clientHintShuffle4[seed%len(clientHintShuffle4)][:]
	}
}

func getUngreasedChUa(majorVersion int, forkName string) []string {
	baseChUa := make([]string, 0, 4)
	baseChUa = append(baseChUa, getGreasedChInvalidBrand(majorVersion),
		"\"Chromium\";v=\""+strconv.Itoa(majorVersion)+"\"")
	switch forkName {
	case "chrome":
		baseChUa = append(baseChUa, "\"Google Chrome\";v=\""+strconv.Itoa(majorVersion)+"\"")
	case "edge":
		baseChUa = append(baseChUa, "\"Microsoft Edge\";v=\""+strconv.Itoa(majorVersion)+"\"")
	}
	return baseChUa
}

func getGreasedChUa(majorVersion int, forkName string) string {
	ungreasedCh := getUngreasedChUa(majorVersion, forkName)
	shuffleMap := getGreasedChOrder(len(ungreasedCh), majorVersion)
	shuffledCh := make([]string, len(ungreasedCh))
	for i, e := range shuffleMap {
		shuffledCh[e] = ungreasedCh[i]
	}
	return strings.Join(shuffledCh, ", ")
}

var (
	curlUA                 = "curl/" + curlVersion()
	anchoredFirefoxVersion = strconv.Itoa(firefoxVersion())
	firefoxUA              = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:" + anchoredFirefoxVersion + ".0) Gecko/20100101 Firefox/" + anchoredFirefoxVersion + ".0"
	safariUA               = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/" + safariVersion() + " Safari/605.1.15"

	anchoredChromeVersion = chromeVersion()
	chromeUA              = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + strconv.Itoa(anchoredChromeVersion) + ".0.0.0 Safari/537.36"
	chromeUACH            = getGreasedChUa(anchoredChromeVersion, "chrome")
	msEdgeUA              = chromeUA + "Edg/" + strconv.Itoa(anchoredChromeVersion) + ".0.0.0"
	msEdgeUACH            = getGreasedChUa(anchoredChromeVersion, "edge")
)

func applyMasqueradedHeaders(header http.Header, browser string, variant string) {
	// Browser-specific.
	switch browser {
	case "chrome":
		header["Sec-CH-UA"] = []string{chromeUACH}
		header["Sec-CH-UA-Mobile"] = []string{"?0"}
		header["Sec-CH-UA-Platform"] = []string{"\"Windows\""}
		header["DNT"] = []string{"1"}
		header.Set("User-Agent", chromeUA)
		header.Set("Accept-Language", "en-US,en;q=0.9")
	case "edge":
		header["Sec-CH-UA"] = []string{msEdgeUACH}
		header["Sec-CH-UA-Mobile"] = []string{"?0"}
		header["Sec-CH-UA-Platform"] = []string{"\"Windows\""}
		header["DNT"] = []string{"1"}
		header.Set("User-Agent", msEdgeUA)
		header.Set("Accept-Language", "en-US,en;q=0.9")
	case "firefox":
		header.Set("User-Agent", firefoxUA)
		header["DNT"] = []string{"1"}
		header.Set("Accept-Language", "en-US,en;q=0.5")
	case "safari":
		header.Set("User-Agent", safariUA)
		header.Set("Accept-Language", "en-US,en;q=0.9")
	case "golang":
		// Expose the default net/http header.
		header.Del("User-Agent")
		return
	case "curl":
		header.Set("User-Agent", curlUA)
		return
	}
	// Context-specific.
	switch variant {
	case "nav":
		if header.Get("Cache-Control") == "" {
			switch browser {
			case "chrome", "edge":
				header.Set("Cache-Control", "max-age=0")
			}
		}
		header.Set("Upgrade-Insecure-Requests", "1")
		if header.Get("Accept") == "" {
			switch browser {
			case "chrome", "edge":
				header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/jxl,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
			case "firefox", "safari":
				header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			}
		}
		header.Set("Sec-Fetch-Site", "none")
		header.Set("Sec-Fetch-Mode", "navigate")
		switch browser {
		case "safari":
		default:
			header.Set("Sec-Fetch-User", "?1")
		}
		header.Set("Sec-Fetch-Dest", "document")
		header.Set("Priority", "u=0, i")
	case "ws":
		header.Set("Sec-Fetch-Mode", "websocket")
		switch browser {
		case "safari":
			// Safari is NOT web-compliant here!
			header.Set("Sec-Fetch-Dest", "websocket")
		default:
			header.Set("Sec-Fetch-Dest", "empty")
		}
		header.Set("Sec-Fetch-Site", "same-origin")
		if header.Get("Cache-Control") == "" {
			header.Set("Cache-Control", "no-cache")
		}
		if header.Get("Pragma") == "" {
			header.Set("Pragma", "no-cache")
		}
		if header.Get("Accept") == "" {
			header.Set("Accept", "*/*")
		}
	case "fetch":
		header.Set("Sec-Fetch-Mode", "cors")
		header.Set("Sec-Fetch-Dest", "empty")
		header.Set("Sec-Fetch-Site", "same-origin")
		if header.Get("Priority") == "" {
			switch browser {
			case "chrome", "edge":
				header.Set("Priority", "u=1, i")
			case "firefox":
				header.Set("Priority", "u=4")
			case "safari":
				header.Set("Priority", "u=3, i")
			}
		}
		if header.Get("Cache-Control") == "" {
			header.Set("Cache-Control", "no-cache")
		}
		if header.Get("Pragma") == "" {
			header.Set("Pragma", "no-cache")
		}
		if header.Get("Accept") == "" {
			header.Set("Accept", "*/*")
		}
	}
}

// TryDefaultHeadersWith mirrors Xray's utils.TryDefaultHeadersWith.
// A user-supplied User-Agent of "chrome"/"firefox"/"safari"/"edge"/"curl"/
// "golang" is a SELECTOR, not a literal value.
func TryDefaultHeadersWith(header http.Header, variant string) {
	if len(header.Values("User-Agent")) < 1 {
		applyMasqueradedHeaders(header, "chrome", variant)
	} else {
		switch header.Get("User-Agent") {
		case "chrome":
			applyMasqueradedHeaders(header, "chrome", variant)
		case "firefox":
			applyMasqueradedHeaders(header, "firefox", variant)
		case "safari":
			applyMasqueradedHeaders(header, "safari", variant)
		case "edge":
			applyMasqueradedHeaders(header, "edge", variant)
		case "curl":
			applyMasqueradedHeaders(header, "curl", variant)
		case "golang":
			applyMasqueradedHeaders(header, "golang", variant)
		}
	}
}
