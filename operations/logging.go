package operations

import (
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/send"
)

func SetupLogging(format string, fileName string) error {
	var sender send.Sender
	var err error

	switch format {
	case "stdout":
		sender = send.NewNative()
	case "file":
		sender, err = send.MakeFileLogger(fileName)
	case "json-stdout":
		sender = send.MakeJSONConsoleLogger()
	case "json-file":
		sender, err = send.MakeJSONFileLogger(fileName)
	// case "systemd":
	// 	sender = setupSystemdLogging()
	case "syslog":
		sender = setupSyslogLogging()
	default:
		grip.Warningf("no supported output format '%s' writing log messages to standard output", format)
		sender = send.NewNative()
	}

	if err != nil {
		return errors.Wrapf(err, "log type %s is not configured", format)
	}

	grip.SetSender(sender)
	return nil
}
