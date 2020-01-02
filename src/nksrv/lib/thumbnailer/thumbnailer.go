package thumbnailer

import (
	"os"

	"nksrv/lib/fstore"
	"nksrv/lib/ftypes"
	. "nksrv/lib/logx"
)

// config
type ThumbConfig struct {
	// thumbnail box
	Width  int
	Height int
	// special audio files cover thumbnailing
	AudioWidth  int
	AudioHeight int
	// rest
	Color     string // background color if needs to be hardcoded
	Grayscale bool   // makes images gray
}

// plan: name + config
type ThumbPlan struct {
	Name string
	ThumbConfig
}

// execution: thumbnailer + name + config
type ThumbExec struct {
	Thumbnailer
	ThumbPlan
}

type FileInfo struct {
	Kind         ftypes.FTypeT
	DetectedType string
	Attrib       map[string]interface{}
}

type ThumbContent struct {
	FullTmpName string
	Suffix      string
}

type ThumbResult struct {
	FI FileInfo // info extracted from file
	Width, Height int // width and height of generated thumb(s)
	DBSuffix string // suffix to be stored in database; may be template
	CF ThumbContent // first
	CE []ThumbContent // extra, if any
}

type ThumbnailerBuilder interface {
	BuildThumbnailer(fs *fstore.FStore, lx LoggerX) (Thumbnailer, error)
}

type Thumbnailer interface {
	// ThumbProcess tries to thumbnail f. Closes f after it's done.
	ThumbProcess(
		f *os.File, ext, mimeType string, cfg ThumbConfig) (
		res ThumbResult, err error)
}
