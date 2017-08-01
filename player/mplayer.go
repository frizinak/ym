package player

func NewMPlayer() *GenericPlayer {
	return &GenericPlayer{
		cmd: "mplayer",
		paramMap: map[Param][]string{
			PARAM_NO_VIDEO: {"-vo", "null"},
			PARAM_SILENT:   {"-really-quiet"},
		},
		commandMap: map[Command][]byte{
			CMD_PAUSE: []byte(" "),
			CMD_NEXT:  []byte("\033[C"),
			CMD_PREV:  []byte("\033[D"),
		},
	}
}
