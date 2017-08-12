package audio

func NewMEncoder() *GenericExtractor {
	return &GenericExtractor{
		ext: "mp3",
		cmd: "mencoder",
		args: []string{
			"-",
			"-really-quiet",
			"-ovc", "frameno",
			"-oac", "mp3lame",
			"-o", "-",
		},
	}
}
