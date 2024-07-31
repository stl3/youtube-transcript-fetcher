# YouTube Transcript Fetcher

YouTube Transcript Fetcher is a command-line tool written in Go that allows you to download and save transcripts from YouTube videos. It supports multiple languages and provides various options for customizing the output.

## Features

- Fetch transcripts from YouTube videos using video ID or URL
- Support for multiple languages
- Customizable output format
- Error handling for various scenarios (video unavailable, transcripts disabled, etc.)

## Installation

### Prerequisites

- Go 1.15 or higher

### Steps

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/youtube-transcript-fetcher.git
   ```

2. Navigate to the project directory:
   ```
   cd youtube-transcript-fetcher
   ```

3. Build the program:
   ```
   go build -o youtube-transcript-fetcher
   ```

## Usage

Basic usage:

```
./youtube-transcript-fetcher -videoId=VIDEO_ID_OR_URL
```

To see all available options:

```
./youtube-transcript-fetcher -h
```

### Options

- `-videoId`: YouTube video ID or URL (required)
- `-lang`: Language code for the transcript (default: "en")
- `-output`: Output file path (default: "transcript.txt")
- `-showText`: Show transcript text (default: true)
- `-showDuration`: Show transcript duration (default: true)
- `-showOffset`: Show transcript offset (default: true)
- `-showLang`: Show transcript language (default: true)
- `-disableAll`: Disable all transcript output fields (default: false)
- `-noTextPrefix`: Disable prefix 'Text: ' in front of transcript text (default: false)

### Examples

1. Fetch English transcript for a video:
   ```
   ./youtube-transcript-fetcher -videoId=dQw4w9WgXcQ -lang=en -output=transcript.txt
   ```

2. Fetch Spanish transcript and only show text:
   ```
   ./youtube-transcript-fetcher -videoId=https://www.youtube.com/watch?v=dQw4w9WgXcQ -lang=es -showDuration=false -showOffset=false -showLang=false
   ```

3. Fetch transcript without 'Text: ' prefix:
   ```
   ./youtube-transcript-fetcher -videoId=dQw4w9WgXcQ -showDuration=false -showOffset=false -showLang=false -noTextPrefix=true
   ```

## Error Handling

The program handles various error scenarios, including:

- Video unavailable
- Transcripts disabled for the video
- Requested language not available
- Too many requests (captcha required)

In case of an error, an appropriate message will be displayed.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- This project was inspired by the need for a simple, command-line tool to fetch YouTube transcripts.
- Thanks to the Go community for providing excellent libraries and documentation.

