package mailer

import (
	"log"
	"path"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type smtpConfig struct {
	Server   string
	Port     int
	Email    string
	Password string
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

// Load SMTP server realetd config
func (c *smtpConfig) load() {
	// fmt.Println(basepath)
	if _, err := toml.DecodeFile(path.Join(basepath, "config.toml"), &c); err != nil {
		log.Fatal(err)
	}
}
