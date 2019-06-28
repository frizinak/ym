package player

func NewFFPlay() *GenericPlayer {
	return &GenericPlayer{
		cmd: "ffplay",
		paramMap: map[Param][]string{
			ParamNoVideo: {"-vn", "-nodisp"},
			ParamSilent:  {"-loglevel", "quiet"},
		},
		commandMap: map[Command][]byte{
			CmdPause: []byte(" "),
			CmdNext:  []byte("\033[C"),
			CmdPrev:  []byte("\033[D"),
		},
	}
}
