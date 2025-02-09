// Package Gobalt provides a go way to communicate with https://cobalt.tools servers.

package gobalt

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mcuadros/go-version"
)

var (
	CobaltApi = "https://cobalt-api.kwiatekmiki.com" //Override this value to use your own cobalt instance. See https://instances.cobalt.best for alternatives from the main instance.
	Client    = http.Client{
		Timeout: 10 * time.Second,
	} //This allows you to modify the HTTP Client used in requests. This Client will be re-used.
	useragent = fmt.Sprintf("gobalt/2.0.9 (+https://github.com/lostdusty/gobalt/v2; go/%v; %v/%v)", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	ApiKey    = os.Getenv("COBALT_API_KEY") //Some instances need an API key to work, set it here. Default is from environment variable `COBALT_API_KEY`.
)

// ServerInfo is the struct used in the function CobaltServerInfo(). It contains two sub-structs: Cobalt and Git
type ServerInfo struct {
	Cobalt CobaltServerInformation `json:"cobalt"`
	Git    CobaltGitInformation    `json:"git"`
}

// This is ServerInfo.Cobalt struct, it contains information about the cobalt backend running on the server.
type CobaltServerInformation struct {
	Version       string   `json:"version"`       //Cobalt version running.
	URL           string   `json:"url"`           //Backend URL of the cobalt server.
	StartTime     string   `json:"startTime"`     //Time when the server started in Unix miliseconds.
	DurationLimit int      `json:"durationLimit"` //Maximum media lenght you can download in seconds. 10800 seconds = 3 hours.
	Services      []string `json:"services"`      //List of configured/enabled services on the instance.
}

// This is ServerInfo.Git struct, it contains informtions about the git commit (from cobalt) the server is using.
type CobaltGitInformation struct {
	Branch string `json:"branch"` //Git branch the cobalt instance is using.
	Commit string `json:"commit"` //Git commit the cobalt instance is using.
	Remote string `json:"remote"` //Git repository name used by the cobalt instance.
}

// CobaltServerInfo(api) gets the server information and returns ServerInfo struct.
//
// This function is called before Run() to check if the cobalt server used is reachable.
// If you can't contact the main server, try using another instance using GetCobaltinstances().
func CobaltServerInfo(api string) (*ServerInfo, error) {
	if !strings.HasPrefix(api, "http") {
		api = "http://" + api
	}
	//Parse url before testing, sanity check
	parseApiUrl, err := url.Parse(api)
	if err != nil {
		return nil, fmt.Errorf("net/url failed to parse provided url, check it and try again (details: %v, url: %v)", err, api)
	}

	if parseApiUrl.Scheme == "" {
		parseApiUrl.Scheme = "https"
	}

	//Check if the server is reachable
	res, err := genericHttpRequest(parseApiUrl.String(), http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	jsonbody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var serverResponse ServerInfo
	err = json.Unmarshal(jsonbody, &serverResponse)
	if err != nil {
		return nil, err
	}

	return &serverResponse, nil
}

//Server info end

/* Download settings structs and types */

// Struct Settings contains changable options that you can change before download. An URL MUST be set before calling gobalt.Run(Settings).
type Settings struct {
	Url                   string       `json:"url"`                   //Any URL from bilibili.com, instagram, pinterest, reddit, rutube, soundcloud, streamable, tiktok, tumblr, twitch clips, twitter/x, vimeo, vine archive, vk or youtube (as long it's configured on the instance).
	Mode                  downloadMode `json:"downloadMode"`          //Mode to download the videos, either Auto, Audio or Mute. Default: Auto
	Proxy                 bool         `json:"alwaysProxy"`           //Tunnel downloaded file thru cobalt, bypassing potential restrictions and protecting your identity and privacy. Default: false
	AudioBitrate          int          `json:"audioBitrate,string"`   //Audio Bitrate settings. Values: 320Kbps, 256Kbps, 128Kbps, 96Kbps, 64Kbps or 8Kbps. Default: 128
	AudioFormat           audioCodec   `json:"audioFormat"`           //"Best", .mp3, .opus, .ogg or .wav. If not specified will default to "Best".
	FilenameStyle         pattern      `json:"filenameStyle"`         //"Classic", "Basic", "Pretty" or "Nerdy". Default is "Basic".
	DisableMetadata       bool         `json:"disableMetadata"`       //Don't include file metadata. Default: false
	TikTokH265            bool         `json:"tiktokH265"`            //Allows downloading TikTok videos in 1080p at cost of compatibility. Default: false
	TikTokFullAudio       bool         `json:"tiktokFullAudio"`       //Enables download of original sound used in a TikTok video. Default: false
	TwitterConvertGif     bool         `json:"twitterGif"`            //Changes whether twitter gifs should be converted to .gif (Twitter gifs are usually looping .mp4s). Default: true
	VideoQuality          int          `json:"videoQuality,string"`   //144p to 2160p (4K), if not specified will default to 1080p.
	YoutubeDubbedAudio    bool         `json:"youtubeDubBrowserLang"` //Downloads the YouTube dubbed audio according to the value set in YoutubeDubbedLanguage (and if present). Default is English (US). Follows the ISO 639-1 standard.
	YoutubeDubbedLanguage string       `json:"youtubeDubLang"`        //Language code to download the dubbed audio, Default is "en".
	YoutubeHLS            bool         `json:"youtubeHLS"`            //Enables downloading YouTube videos using HLS streams. (Less prone to fail) Default: true
	YoutubeVideoFormat    videoCodecs  `json:"youtubeVideoCodec"`     //Which video format to download from YouTube, see videoCodecs type for details.
}

type downloadMode string

const (
	Audio downloadMode = "audio" //Download only the audio.
	Auto  downloadMode = "auto"  //Auto mode, audio + video (if video is present).
	Mute  downloadMode = "mute"  //Downloads only the video, no audio.
)

type videoCodecs string

const (
	H264 videoCodecs = "h264" //Default codec that is supported everywhere. Recommended for social media/phones, but tops up at 1080p.
	AV1  videoCodecs = "av1"  //Recent codec, supports 8K/HDR. Generally less supported by media players, social media, etc.
	VP9  videoCodecs = "vp9"  //Best quality codec with higher bitrate (preserve most detail), goes up to 4K and supports HDR.
)

type audioCodec string

const (
	Best audioCodec = "best" //When "best" format is selected, you get audio the way it is on service's side. it's not re-encoded.
	Opus audioCodec = "opus" //Re-encodes the audio using Opus codec. It's a lossy codec with low complexity. Works in Android 10+, Windows 10 1809+, MacOS High Sierra/iOS 17+.
	Ogg  audioCodec = "ogg"  //Re-encodes to ogg, an older lossy audio codec. Should work everywhere.
	Wav  audioCodec = "wav"  //Re-encodes to wav, an even older format. Good compatibility for older systems, like Windows 98. Tops up at 4GiB.
	MP3  audioCodec = "mp3"  //Re-encodes to mp3, the format used basically everywhere. Lossy audio, but generally good player/social media support. Can degrade quality as time passes.
)

type pattern string

const (
	Classic pattern = "classic" //Looks like this: youtube_yPYZpwSpKmA_1920x1080_h264.mp4 | audio: youtube_yPYZpwSpKmA_audio.mp3
	Basic   pattern = "basic"   //Looks like: Video Title (1080p, h264).mp4 | audio: Audio Title - Audio Author.mp3
	Nerdy   pattern = "nerdy"   //Looks like this: Video Title (1080p, h264, youtube, yPYZpwSpKmA).mp4 | audio: Audio Title - Audio Author (soundcloud, 1242868615).mp3
	Pretty  pattern = "pretty"  //Looks like: Video Title (1080p, h264, youtube).mp4 | audio: Audio Title - Audio Author (soundcloud).mp3
)

// This function creates the Settings struct with these default values:
//
//   - Url: "" (empty)
//   - YoutubeVideoFormat: `H264`
//   - VideoQuality: `1080`
//   - AudioFormat: `Best`
//   - AudioBitrate: `128`
//   - FilenameStyle: `Basic`
//   - TwitterConvertGif: `true`
//   - Mode: `Auto`
//
// You MUST set an url before calling Run().
func CreateDefaultSettings() Settings {
	options := Settings{
		Url:                   "",
		YoutubeVideoFormat:    H264,
		VideoQuality:          1080,
		AudioFormat:           Best,
		AudioBitrate:          128,
		FilenameStyle:         Basic,
		TwitterConvertGif:     true,
		Mode:                  Auto,
		YoutubeDubbedLanguage: "en",
		YoutubeHLS:            true,
	}
	return options
}

// Cobalt response to your request
type CobaltResponse struct {
	Status string      `json:"status"` //4 possible status. Error = Something went wrong, see CobaltResponse.Error.Code | Tunnel or Redirect = Everything is right. | Picker = Multiple media, see CobaltResponse.Picker.
	Picker *[]struct { //This is an array of items, each containing the media type, url to download and thumbnail. May be <NIL> if the status is not picker.
		Type  string `json:"type"`  //Type of the media, either photo, video or gif
		URL   string `json:"url"`   //Url to download.
		Thumb string `json:"thumb"` //Media preview url, optional.
	} `json:"picker"`
	URL      string     `json:"url"`      //Returns the download link. If the status is picker this field will be empty. Direct link to a file or a link to cobalt's live render.
	Filename string     `json:"filename"` //Various text, mostly used for errors.
	Error    *Error     `json:"error"`    //Error information, may be <NIL> if theres no error.
	Server   ServerInfo //Server information, see ServerInfo struct.
}

type Error struct {
	Code    string  `json:"code"`    // Machine-readable error code explaining the failure reason.
	Context Context `json:"context"` //(optional) container for providing more context.
}

var (
	// Map machine-readable error codes to human-readable error messages.
	ErrDescriptions = map[string]string{
		"error.api.auth.key.invalid":          "no api key was provided, please provide an api key to use this server",
		"error.api.auth.jwt.missing":          "this server supports API keys, but you didn't provide one",
		"error.api.auth.jwt.invalid":          "the api key you provided is invalid",
		"error.api.auth.turnstile.missing":    "this instance uses turnstile",
		"error.api.auth.turnstile.invalid":    "the turnstile token you provided is invalid",
		"error.api.rate_exceeded":             "you are making too many requests! try again later",
		"error.api.capacity":                  "this cobalt server can't process your request right now",
		"error.api.generic":                   "something went wrong on the server side, try again, and if it still doesn't work, contact the server owner",
		"error.api.unknown_response":          "the server returned an unknown response",
		"error.api.service.unsupported":       "this cobalt server doesn't support the service you're trying to use",
		"error.api.service.disabled":          "the service you're trying to download is disabled on this server",
		"error.api.link.invalid":              "the link you provided is invalid, is this a valid link?",
		"error.api.link.unsupported":          "the link you provided is supported, but cobalt couldn't recognize it, is your link correct?",
		"error.api.fetch.fail":                "an unknown error occurred while fetching the media, does this link works?",
		"error.api.fetch.critical":            "the service you're trying to download is returning something unexpected, try again later",
		"error.api.fetch.empty":               "the service you're trying to download is returning an empty response, try again later",
		"error.api.fetch.rate":                "the cobalt server got rate-limited by the service you're trying to download, try again later",
		"error.api.content.too_long":          "the media you're trying to download is too long, try downloading a shorter video",
		"error.api.content.video.unavailable": "either the video you're trying to download is region-locked, or the service is blocking cobalt",
		"error.api.content.video.live":        "the video you're trying to download is live, and cobalt can't download live videos",
		"error.api.content.video.age":         "the video you're trying to download is age-restricted, and cobalt can't download age-restricted videos",
		"error.api.content.video.private":     "the video you're trying to download is private, make sure it's public or unlisted",
		"error.api.content.video.region":      "the video you're trying to download is region restricted",
		"error.api.youtube.codec":             "try using a different codec, this video doesn't have the codec you're trying to download",
		"error.api.youtube.decipher":          "cobalt couldn't decipher the video, try again later",
		"error.api.youtube.login":             "youtube marked the processing server as a bot, tell the owner to check cookies",
		"error.api.youtube.token_expired":     "the youtube token expired, try again in a few seconds, but if it still doesn't work, tell the instance owner about this error",
		"error.api.youtube.no_hls_streams":    "the video you're trying to download doesn't have any HLS streams, try other settings",
		"error.net.failed":                    "unable to connect to the cobalt server, check your internet connection, the server status, and try again",
		"error.net.generic":                   "an unknown error occurred while connecting to the cobalt server.",
		"error.net.invalid_response":          "the cobalt server returned an invalid response, try again later",
	}
)

// ResolveError(error) returns a human-readable error message from the error code.
func ResolveError(code error) string {
	if val, ok := ErrDescriptions[code.Error()]; ok {
		return fmt.Sprintf("%v (%v)", val, code.Error())
	}
	return code.Error()
}

type Context struct {
	Service string `json:"service"`         //What service failed.
	Limit   int    `json:"limit,omitempty"` //Number providing the ratelimit maximum number of requests, or maximum downloadable video duration
}

// Run(gobalt.Settings) sends the request to the provided cobalt api and returns the server response (gobalt.CobaltResponse) and error, use this to download something AFTER setting your desired configuration.
// Use ErrDescriptions to get a human-readable error message from the error code.
func Run(options Settings) (*CobaltResponse, error) {
	//Check if an url is set.
	if options.Url == "" {
		return nil, errors.New("no url was provided to download")
	}

	//Do a basic check to see if the server is online and handling requests
	//Also add to CobaltResponse the server information.
	_, err := CobaltServerInfo(CobaltApi)
	if err != nil {
		return nil, fmt.Errorf("error.net.generic: %v", err)
	}

	jsonBody, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("error.net.invalid_response")
	}

	req, err := http.NewRequest(http.MethodPost, CobaltApi, strings.NewReader(string(jsonBody)))
	req.Header.Add("User-Agent", useragent)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Api-Key "+ApiKey)
	if err != nil {
		return nil, err
	}

	res, err := Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error.net.failed")
	}
	defer res.Body.Close()

	jsonbody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error.net.invalid_response")
	}

	var media CobaltResponse
	err = json.Unmarshal(jsonbody, &media)
	if err != nil {
		return nil, fmt.Errorf("error.net.invalid_response")
	}

	if media.Status == "error" {
		return nil, fmt.Errorf("%v", media.Error.Code)
	}

	return &media, nil
}

/* End of: Download settings structs and types */

//Cobalt response end

// CobaltInstance is a struct that contains information about a cobalt instance.
type CobaltInstance []struct {
	API      string       `json:"api"`
	Branch   string       `json:"branch"`
	Commit   string       `json:"commit"`
	Cors     bool         `json:"cors"`
	Frontend string       `json:"frontend"`
	Name     string       `json:"name"`
	Nodomain bool         `json:"nodomain"`
	Online   OnlineStatus `json:"online"`
	Protocol string       `json:"protocol"`
	Score    int          `json:"score"`
	//Services EnabledServices `json:"services"`
	Trust   int    `json:"trust"`
	Version string `json:"version"`
}
type OnlineStatus struct {
	API      bool `json:"api"`
	Frontend bool `json:"frontend"`
}
type EnabledServices struct {
	Bilibili      bool   `json:"bilibili"`
	BilibiliTv    bool   `json:"bilibili_tv"`
	Bluesky       bool   `json:"bluesky"`
	Dailymotion   bool   `json:"dailymotion"`
	Facebook      bool   `json:"facebook"`
	Instagram     bool   `json:"instagram"`
	Loom          bool   `json:"loom"`
	Odnoklassniki bool   `json:"odnoklassniki"`
	Pinterest     bool   `json:"pinterest"`
	Reddit        bool   `json:"reddit"`
	Rutube        bool   `json:"rutube"`
	Snapchat      bool   `json:"snapchat"`
	Soundcloud    bool   `json:"soundcloud"`
	Streamable    bool   `json:"streamable"`
	Tiktok        bool   `json:"tiktok"`
	Tumblr        bool   `json:"tumblr"`
	Twitch        bool   `json:"twitch"`
	Twitter       bool   `json:"twitter"`
	Vimeo         bool   `json:"vimeo"`
	Vine          bool   `json:"vine"`
	Vk            bool   `json:"vk"`
	Youtube       string `json:"youtube"`
	YoutubeMusic  string `json:"youtube_music"`
	YoutubeShorts string `json:"youtube_shorts"`
}

// GetCobaltInstances makes a request to instances.cobalt.best and returns a list of all online cobalt instances.
func GetCobaltInstances() (CobaltInstance, error) {
	res, err := genericHttpRequest("https://instances.cobalt.best/api/instances.json", http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	jsonbody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var listOfCobaltInstances CobaltInstance
	err = json.Unmarshal(jsonbody, &listOfCobaltInstances)
	if err != nil {
		return nil, err
	}

	parseModernInstances := make(CobaltInstance, 0)
	for _, v := range listOfCobaltInstances {
		if version.Compare(v.Version, "10.0.0", ">=") {
			parseModernInstances = append(parseModernInstances, v)
		}

	}

	return parseModernInstances, nil
	//return listOfCobaltInstances, nil
}

// Deprecated: Cobalt response returns the file name and size.
type MediaInfo struct {
	Size uint   //Media size in bytes.
	Name string //Media name.
	Type string //Mime type of the media.
}

// ProcessMedia(url) attempts to fetch the file size, mime type and name.
// Deprecated: Cobalt response returns the file name and size.
func ProcessMedia(url string) (*MediaInfo, error) {
	req, err := genericHttpRequest(url, http.MethodHead, nil)
	if err != nil {
		return nil, err
	}
	_, parsefilename, err := mime.ParseMediaType(req.Header.Get("Content-Disposition"))
	filename := parsefilename["filename"]
	if err != nil {
		filename = path.Base(req.Request.URL.Path)
	}
	size := req.Header.Get("Content-Length")
	if size == "" {
		size = "0"
	}
	parseSize, err := strconv.Atoi(size)
	if err != nil {
		return nil, err
	}

	return &MediaInfo{
		Size: uint(parseSize),
		Name: filename,
		Type: req.Header.Get("Content-Type"),
	}, nil
}

// This slice will contain urls of Youtube videos
type Playlist []string

// Function GetYoutubePlaylist(string) gets an Youtube playlist has parameter, and returns a slice []Playlist with the urls of the playlist.
func GetYoutubePlaylist(playlist string) (Playlist, error) {
	//Parse param url
	newYoutubePlaylistUrl, err := url.Parse(playlist)
	if err != nil {
		return nil, err
	}

	getUrls, err := genericHttpRequest(fmt.Sprintf("https://playlist.kwiatekmiki.pl/api/getvideos?url=%v", newYoutubePlaylistUrl.String()), http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	if getUrls.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get playlists: %v", getUrls.Status)
	}

	unmarshalBody, err := io.ReadAll(getUrls.Body)
	if err != nil {
		return nil, err
	}

	var list Playlist
	err = json.Unmarshal(unmarshalBody, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// Function to do generic, less complex http requests, to avoid code repetitions. Internal use of the library only.
func genericHttpRequest(url, method string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, url, body)
	request.Header.Add("User-Agent", useragent)

	if err != nil {
		return nil, err
	}

	response, err := Client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with %v", response.Status)
	}

	return response, nil
}
