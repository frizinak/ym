package audio

func NewFFMPEG() *GenericExtractor {
	return &GenericExtractor{
		cmd: "ffmpeg",
		args: []string{
			"-i", "-",
			"-vn",
			"-acodec", "copy",
			"-f", "adts",
			"-",
		},
	}
}
