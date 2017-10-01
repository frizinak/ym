package audio

func NewFFMPEG() *GenericExtractor {
	return &GenericExtractor{
		cmd: "ffmpeg",
		args: []string{
			"-i", "-",
			"-vn",
			//"-acodec", "copy",
			"-af", "silenceremove=1:0:0:1:0:0",
			"-f", "adts",
			"-",
		},
	}
}
