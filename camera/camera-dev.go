// +build dev

package camera

import (
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.Warnln("DEVELOPMENT MODE")
	developmentMode = true
}
