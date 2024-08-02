package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const RE_YOUTUBE = `(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/\s]{11})`
const USER_AGENT = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36`
const RE_XML_TRANSCRIPT = `<text start="([^"]*)" dur="([^"]*)">([^<]*)<\/text>`

type YoutubeTranscriptError struct {
	Message string
}

func (e *YoutubeTranscriptError) Error() string {
	return fmt.Sprintf("[YoutubeTranscript] 🚨 %s", e.Message)
}

type YoutubeTranscriptTooManyRequestError struct {
	YoutubeTranscriptError
}

type YoutubeTranscriptVideoUnavailableError struct {
	YoutubeTranscriptError
	VideoID string
}

type YoutubeTranscriptDisabledError struct {
	YoutubeTranscriptError
	VideoID string
}

type YoutubeTranscriptNotAvailableError struct {
	YoutubeTranscriptError
	VideoID string
}

type YoutubeTranscriptNotAvailableLanguageError struct {
	YoutubeTranscriptError
	Lang           string
	AvailableLangs []string
	VideoID        string
}

type TranscriptConfig struct {
	Lang string
}

type TranscriptResponse struct {
	Text     string
	Duration float64
	Offset   float64
	Lang     string
}

type YoutubeTranscript struct{}

func (yt *YoutubeTranscript) FetchTranscript(videoId string, config *TranscriptConfig) ([]TranscriptResponse, string, error) {
	identifier, err := retrieveVideoId(videoId)
	if err != nil {
		return nil, "", err
	}

	videoPageURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", identifier)
	videoPageResponse, err := http.Get(videoPageURL)
	if err != nil {
		return nil, "", err
	}
	defer videoPageResponse.Body.Close()

	videoPageBody, err := ioutil.ReadAll(videoPageResponse.Body)
	if err != nil {
		return nil, "", err
	}

	// Extract video title
	titleRegex := regexp.MustCompile(`<title>(.+?) - YouTube</title>`)
	titleMatch := titleRegex.FindSubmatch(videoPageBody)
	var videoTitle string
	if len(titleMatch) > 1 {
		videoTitle = string(titleMatch[1])
	} else {
		videoTitle = "Untitled Video"
	}

	splittedHTML := strings.Split(string(videoPageBody), `"captions":`)
	if len(splittedHTML) <= 1 {
		if strings.Contains(string(videoPageBody), `class="g-recaptcha"`) {
			return nil, "", &YoutubeTranscriptTooManyRequestError{YoutubeTranscriptError{Message: "YouTube is receiving too many requests from this IP and now requires solving a captcha to continue"}}
		}
		if !strings.Contains(string(videoPageBody), `"playabilityStatus":`) {
			return nil, "", &YoutubeTranscriptVideoUnavailableError{YoutubeTranscriptError{Message: fmt.Sprintf("The video is no longer available (%s)", videoId)}, videoId}
		}
		return nil, "", &YoutubeTranscriptDisabledError{YoutubeTranscriptError{Message: fmt.Sprintf("Transcript is disabled on this video (%s)", videoId)}, videoId}
	}

	var captions struct {
		PlayerCaptionsTracklistRenderer struct {
			CaptionTracks []struct {
				BaseURL      string `json:"baseUrl"`
				LanguageCode string `json:"languageCode"`
			} `json:"captionTracks"`
		} `json:"playerCaptionsTracklistRenderer"`
	}

	captionsData := splittedHTML[1][:strings.Index(splittedHTML[1], ",\"videoDetails")]
	err = json.Unmarshal([]byte(captionsData), &captions)
	if err != nil {
		fmt.Println("Error unmarshalling captions data:", err)
		return nil, "", &YoutubeTranscriptDisabledError{YoutubeTranscriptError{Message: fmt.Sprintf("Transcript is disabled on this video (%s)", videoId)}, videoId}
	}

	if len(captions.PlayerCaptionsTracklistRenderer.CaptionTracks) == 0 {
		return nil, "", &YoutubeTranscriptNotAvailableError{YoutubeTranscriptError{Message: fmt.Sprintf("No transcripts are available for this video (%s)", videoId)}, videoId}
	}

	var transcriptURL string
	if config != nil && config.Lang != "" {
		for _, track := range captions.PlayerCaptionsTracklistRenderer.CaptionTracks {
			if track.LanguageCode == config.Lang {
				transcriptURL = track.BaseURL
				break
			}
		}
		if transcriptURL == "" {
			availableLangs := make([]string, len(captions.PlayerCaptionsTracklistRenderer.CaptionTracks))
			for i, track := range captions.PlayerCaptionsTracklistRenderer.CaptionTracks {
				availableLangs[i] = track.LanguageCode
			}
			return nil, "", &YoutubeTranscriptNotAvailableLanguageError{
				YoutubeTranscriptError{Message: fmt.Sprintf("No transcripts are available in %s for this video (%s). Available languages: %s", config.Lang, videoId, strings.Join(availableLangs, ", "))},
				config.Lang, availableLangs, videoId,
			}
		}
	} else {
		transcriptURL = captions.PlayerCaptionsTracklistRenderer.CaptionTracks[0].BaseURL
	}

	fmt.Println("Transcript URL:", transcriptURL) // Debugging line

	transcriptResponse, err := http.Get(transcriptURL)
	if err != nil {
		return nil, "", &YoutubeTranscriptNotAvailableError{YoutubeTranscriptError{Message: fmt.Sprintf("No transcripts are available for this video (%s)", videoId)}, videoId}
	}
	defer transcriptResponse.Body.Close()

	transcriptBody, err := ioutil.ReadAll(transcriptResponse.Body)
	if err != nil {
		return nil, "", err
	}

	re := regexp.MustCompile(RE_XML_TRANSCRIPT)
	matches := re.FindAllStringSubmatch(string(transcriptBody), -1)
	var results []TranscriptResponse
	for _, match := range matches {
		duration, _ := strconv.ParseFloat(match[2], 64)
		offset, _ := strconv.ParseFloat(match[1], 64)
		results = append(results, TranscriptResponse{
			Text:     match[3],
			Duration: duration,
			Offset:   offset,
			Lang:     config.Lang,
		})
	}
	return results, videoTitle, nil
}

func retrieveVideoId(videoId string) (string, error) {
	if len(videoId) == 11 {
		return videoId, nil
	}
	re := regexp.MustCompile(RE_YOUTUBE)
	match := re.FindStringSubmatch(videoId)
	if match != nil {
		return match[1], nil
	}
	return "", &YoutubeTranscriptError{Message: "Impossible to retrieve Youtube video ID."}
}

func sanitizeFilename(filename string) string {
	// Replace or remove illegal characters
	re := regexp.MustCompile(`[<>:"/\\|? *]`)
	sanitized := re.ReplaceAllString(filename, "_")

	// Remove leading/trailing spaces and dots
	sanitized = strings.Trim(sanitized, " .")

	// Limit the length to 200 characters
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}

	return sanitized
}

func main() {
	// Define command line flags
	videoId := flag.String("videoId", "", "YouTube video ID or URL")
	lang := flag.String("lang", "en", "Language code for the transcript")
	output := flag.String("output", "", "Output file path")
	showText := flag.Bool("showText", true, "Show transcript text")
	showDuration := flag.Bool("showDuration", false, "Show transcript duration")
	showOffset := flag.Bool("showOffset", false, "Show transcript offset")
	showLang := flag.Bool("showLang", false, "Show transcript language")
	disableAll := flag.Bool("disableAll", false, "Disable all transcript output fields")
	noTextPrefix := flag.Bool("noTextPrefix", true, "Disable prefix 'Text: ' in front of transcript text")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -videoId=dQw4w9WgXcQ -lang=en -output=transcript.txt\n", os.Args[0])
	}

	// Parse command line flags
	flag.Parse()

	// If no arguments are provided or -h/--help is used, print usage and exit
	if len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
		flag.Usage()
		os.Exit(0)
	}

	// Validate required flags
	if *videoId == "" {
		fmt.Println("Error: videoId is required")
		flag.Usage()
		os.Exit(1)
	}

	// Disable all fields if disableAll is true
	if *disableAll {
		*showText = false
		*showDuration = false
		*showOffset = false
		*showLang = false
	}

	yt := &YoutubeTranscript{}
	transcripts, videoTitle, err := yt.FetchTranscript(*videoId, &TranscriptConfig{Lang: *lang})
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Determine output filename
	var outputFilename string
	if *output == "" {
		sanitizedTitle := sanitizeFilename(videoTitle)
		outputFilename = sanitizedTitle + ".txt"
	} else {
		outputFilename = *output
	}

	// Create or open the output file
	file, err := os.Create(outputFilename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	defer file.Close()

	// Write transcripts to the file
	for _, transcript := range transcripts {
		if *showText {
			prefix := "Text: "
			suffix := "\n"
			if *noTextPrefix {
				prefix = ""
				suffix = ""
			}
			_, err := file.WriteString(fmt.Sprintf("%s%s%s", prefix, transcript.Text, suffix))
			if err != nil {
				fmt.Println("Error writing to file:", err)
				os.Exit(1)
			}
		}
		if *showDuration {
			_, err := file.WriteString(fmt.Sprintf("Duration: %.2f\n", transcript.Duration))
			if err != nil {
				fmt.Println("Error writing to file:", err)
				os.Exit(1)
			}
		}
		if *showOffset {
			_, err := file.WriteString(fmt.Sprintf("Offset: %.2f\n", transcript.Offset))
			if err != nil {
				fmt.Println("Error writing to file:", err)
				os.Exit(1)
			}
		}
		if *showLang {
			_, err := file.WriteString(fmt.Sprintf("Language: %s\n", transcript.Lang))
			if err != nil {
				fmt.Println("Error writing to file:", err)
				os.Exit(1)
			}
		}
		_, err := file.WriteString("\n")
		if err != nil {
			fmt.Println("Error writing to file:", err)
			os.Exit(1)
		}
	}

	fmt.Println("Transcript saved to", outputFilename)
}
