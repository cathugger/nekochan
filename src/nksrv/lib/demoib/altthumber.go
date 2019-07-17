package demoib

import (
	"nksrv/lib/altthumber"
)

type DemoAltThumber struct{}

var _ altthumber.AltThumber = DemoAltThumber{}

func (DemoAltThumber) GetAltThumb(fname string, typ string) (alt string, width uint32, height uint32) {
	return "demoaltthumb.png", 128, 128
}