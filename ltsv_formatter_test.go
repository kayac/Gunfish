package gunfish

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestQuoting(t *testing.T) {
	tf := &LtsvFormatter{}

	checkQuoting := func(q bool, value interface{}) {
		b, _ := tf.Format(logrus.WithField("test", value))
		idx := bytes.Index(b, ([]byte)("test:"))
		cont := bytes.Equal(b[idx+5:idx+6], []byte{'"'})
		if cont != q {
			if q {
				t.Errorf("quoting expected for: %#v", value)
			} else {
				t.Errorf("quoting not expected for: %#v", value)
			}
		}
	}

	checkQuoting(false, "abcd")
	checkQuoting(false, "v1.0")
	checkQuoting(false, "1234567890")
	checkQuoting(true, "/foobar")
	checkQuoting(true, "x y")
	checkQuoting(true, "x,y")
	checkQuoting(false, errors.New("invalid"))
	checkQuoting(true, errors.New("invalid argument"))
}
