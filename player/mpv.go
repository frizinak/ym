package player

func NewMPV() *GenericPlayer {
	return &GenericPlayer{
		cmd:  "mpv",
		args: []string{"--input-terminal", "--terminal"},
		paramMap: map[Param][]string{
			PARAM_NO_VIDEO: {"--no-video"},
			PARAM_SILENT:   {"-really-quiet"},
		},
		commandMap: map[Command][]byte{
			CMD_PAUSE: []byte(" "),
			CMD_NEXT:  []byte("\033[C"),
			CMD_PREV:  []byte("\033[D"),
		},
	}
}
