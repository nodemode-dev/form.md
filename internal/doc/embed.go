package doc

import (
	"net/url"
	"regexp"
	"strings"
)

type provider struct {
	id     string
	re     *regexp.Regexp
	src    func(m []string, raw string) string
	height int
}

var providers = []provider{
	{"youtube", youtubeRE, func(m []string, _ string) string { return "https://www.youtube-nocookie.com/embed/" + m[1] }, 0},
	{"vimeo", vimeoRE, func(m []string, _ string) string { return "https://player.vimeo.com/video/" + m[1] + "?dnt=1" }, 0},
	{"loom", regexp.MustCompile(`^https://(?:www\.)?loom\.com/(?:share|embed)/([a-f0-9]{16,40})`), func(m []string, _ string) string { return "https://www.loom.com/embed/" + m[1] }, 0},
	{"dailymotion", regexp.MustCompile(`^https://(?:www\.)?dailymotion\.com/video/([A-Za-z0-9]{5,12})`), func(m []string, _ string) string { return "https://www.dailymotion.com/embed/video/" + m[1] }, 0},
	{"tiktok", regexp.MustCompile(`^https://(?:www\.)?tiktok\.com/@[\w.-]+/video/(\d{6,25})`), func(m []string, _ string) string { return "https://www.tiktok.com/embed/v2/" + m[1] }, 740},
	{"figma", regexp.MustCompile(`^https://(?:www\.)?figma\.com/(?:file|design|proto|board|slides)/[A-Za-z0-9]{10,40}`), func(_ []string, raw string) string {
		return "https://www.figma.com/embed?embed_host=formmd&url=" + url.QueryEscape(raw)
	}, 450},
	{"codepen", regexp.MustCompile(`^https://codepen\.io/([\w-]{1,40})/(?:pen|full|details)/([A-Za-z0-9]{4,12})`), func(m []string, _ string) string {
		return "https://codepen.io/" + m[1] + "/embed/" + m[2] + "?default-tab=result"
	}, 400},
	{"codesandbox", regexp.MustCompile(`^https://codesandbox\.io/(?:s|embed|p/sandbox)/([\w-]{3,80})`), func(m []string, _ string) string { return "https://codesandbox.io/embed/" + m[1] }, 500},
	{"spotify", regexp.MustCompile(`^https://open\.spotify\.com/(track|album|playlist|episode|show|artist)/([A-Za-z0-9]{10,30})`), func(m []string, _ string) string {
		return "https://open.spotify.com/embed/" + m[1] + "/" + m[2]
	}, 352},
	{"soundcloud", regexp.MustCompile(`^https://(?:www\.)?soundcloud\.com/[\w-]{1,60}/[\w-]{1,120}`), func(_ []string, raw string) string {
		return "https://w.soundcloud.com/player/?url=" + url.QueryEscape(raw)
	}, 166},
	{"gdocs", regexp.MustCompile(`^https://docs\.google\.com/(document|spreadsheets|presentation)/d/([\w-]{20,80})`), func(m []string, _ string) string {
		return "https://docs.google.com/" + m[1] + "/d/" + m[2] + "/preview"
	}, 600},
	{"gforms", regexp.MustCompile(`^https://docs\.google\.com/forms/d/e/([\w-]{20,80})`), func(m []string, _ string) string {
		return "https://docs.google.com/forms/d/e/" + m[1] + "/viewform?embedded=true"
	}, 600},
	{"gdrive", regexp.MustCompile(`^https://drive\.google\.com/file/d/([\w-]{20,80})`), func(m []string, _ string) string { return "https://drive.google.com/file/d/" + m[1] + "/preview" }, 480},
	{"gmaps", regexp.MustCompile(`^https://(?:www\.|maps\.)?google\.[a-z.]{2,6}/maps(?:/place/([^/?#]+)|.*[?&]q=([^&#]+))`), func(m []string, _ string) string {
		q := m[1]
		if q == "" {
			q = m[2]
		}
		return "https://www.google.com/maps?output=embed&q=" + q
	}, 400},
	{"miro", regexp.MustCompile(`^https://miro\.com/app/board/([\w=-]{6,40})`), func(m []string, _ string) string { return "https://miro.com/app/live-embed/" + m[1] + "/" }, 450},
	{"airtable", regexp.MustCompile(`^https://airtable\.com/(?:embed/)?((?:app\w{10,20}/)?shr\w{10,20})`), func(m []string, _ string) string { return "https://airtable.com/embed/" + m[1] }, 533},
	{"typeform", regexp.MustCompile(`^https://[\w-]{1,60}\.typeform\.com/to/[A-Za-z0-9]{4,20}`), func(m []string, _ string) string { return m[0] }, 500},
	{"excalidraw", regexp.MustCompile(`^https://excalidraw\.com/`), func(_ []string, raw string) string { return raw }, 500},
	{"canva", regexp.MustCompile(`^https://(?:www\.)?canva\.com/design/([\w-]{8,20})/([\w-]{8,40})/`), func(m []string, _ string) string {
		return "https://www.canva.com/design/" + m[1] + "/" + m[2] + "/view?embed"
	}, 480},
	{"behance", regexp.MustCompile(`^https://(?:www\.)?behance\.net/gallery/(\d{5,12})`), func(m []string, _ string) string { return "https://www.behance.net/embed/project/" + m[1] + "?ilo0=1" }, 520},
	{"sketchfab", regexp.MustCompile(`^https://sketchfab\.com/(?:3d-models/[\w-]*?-?|models/)([a-f0-9]{32})`), func(m []string, _ string) string { return "https://sketchfab.com/models/" + m[1] + "/embed" }, 480},
	{"pinterest", regexp.MustCompile(`^https://(?:[a-z]{2,3}\.|www\.)?pinterest\.[a-z.]{2,6}/pin/(\d{6,25})`), func(m []string, _ string) string { return "https://assets.pinterest.com/ext/embed.html?id=" + m[1] }, 620},
	{"applemusic", regexp.MustCompile(`^https://music\.apple\.com/([a-z]{2}/(?:album|playlist|song|music-video)/[^?#\s"<>]{1,200})`), func(m []string, _ string) string { return "https://embed.music.apple.com/" + m[1] }, 450},
}

const webHeight = 480

func EmbedProvider(raw string) string {
	for _, p := range providers {
		if p.re.MatchString(raw) {
			return p.id
		}
	}
	return ""
}

func EmbedSrc(raw, kind string) (string, int) {
	for _, p := range providers {
		if m := p.re.FindStringSubmatch(raw); m != nil {
			return p.src(m, raw), p.height
		}
	}
	if kind == "web" {
		if u, err := url.Parse(raw); err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && !strings.ContainsAny(raw, "\"<> ") {
			return raw, webHeight
		}
	}
	return "", 0
}
