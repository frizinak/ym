package player

func NewMPlayer() *GenericPlayer {
	return &GenericPlayer{
		cmd: "mplayer",
		paramMap: map[Param][]string{
			ParamNoVideo: {"-vo", "null"},
			ParamSilent:  {"-really-quiet"},
		},
		commandMap: map[Command][]byte{
			CmdPause:   []byte(" "),
			CmdNext:    []byte("\033[C"),
			CmdPrev:    []byte("\033[D"),
			CmdVolUp:   []byte("0"),
			CmdVolDown: []byte("9"),
		},
	}
}
