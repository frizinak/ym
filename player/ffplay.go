package player

func NewFFPlay() *GenericPlayer {
	return &GenericPlayer{
		cmd: "ffplay",
		paramMap: map[Param][]string{
			PARAM_NO_VIDEO: {"-vn", "-nodisp"},
			PARAM_SILENT:   {"-loglevel", "quiet"},
		},
		commandMap: map[Command][]byte{
			CMD_PAUSE: []byte(" "),
			CMD_NEXT:  []byte("\033[C"),
			CMD_PREV:  []byte("\033[D"),
		},
	}
}
