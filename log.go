package depot

import (
	"io/ioutil"
	"os"

	"github.com/choueric/clog"
)

const (
	logFlag = clog.Ldate | clog.Ltime | clog.Lshortfile | clog.Lcolor
)

var (
	isDebug bool
	dbgLog  = clog.New(ioutil.Discard, "", 0)
)

func SetDebug(d bool) *clog.Logger {
	if d {
		dbgLog = clog.New(os.Stderr, "", logFlag)
	} else {
		dbgLog = clog.New(ioutil.Discard, "", 0)
	}
	return dbgLog
}
