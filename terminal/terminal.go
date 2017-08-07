package terminal

// func Prompt() (input string, err error) {
// 	var line []byte
//
// 	line, _, err = bufio.NewReader(os.Stdin).ReadLine()
// 	input = string(line)
// 	return
// }
//
// func IntPrompt() (input int, err error) {
// 	var line string
// 	line, err = Prompt()
// 	if err != nil {
// 		return
// 	}
//
// 	input, err = strconv.Atoi(line)
// 	return
// }

// Props to nsf/termbox-go
// type winsize struct {
// 	rows    uint16
// 	cols    uint16
// 	xpixels uint16
// 	ypixels uint16
// }
//
// func Dimensions() (int, int) {
// 	var sz winsize
// 	syscall.Syscall(
// 		syscall.SYS_IOCTL,
// 		uintptr(syscall.Stdin),
// 		uintptr(syscall.TIOCGWINSZ),
// 		uintptr(unsafe.Pointer(&sz)),
// 	)
//
// 	return int(sz.cols), int(sz.rows)
// }

// func Clear() {
// 	os.Stdout.WriteString("\033c")
// }
