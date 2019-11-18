package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/spf13/viper"
	try "gopkg.in/matryer/try.v1"
)

const (
	mime = "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
)

func parseTemplate(fileName string, data interface{}) (string, error) {
	t, err := template.ParseFiles(fileName)
	if err != nil {
		return "", err
	}
	buffer := new(bytes.Buffer)
	if err = t.Execute(buffer, data); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// Send email from configured SMTP server
func Send(templateName string, templateData interface{}, subject string, recipent string) error {
	body, err := parseTemplate(templateName, templateData)
	if err != nil {
		return err
	}
	content := "To: " + recipent + "\r\nSubject: " + subject + "\r\n" + mime + "\r\n" + body
	SMTP := fmt.Sprintf("%s:%d", viper.GetString("SMTPServer"), viper.GetInt("SMTPPort"))

	// Retry sending emails
	err = try.Do(func(attempt int) (bool, error) {
		e := smtp.SendMail(SMTP, smtp.PlainAuth("", viper.GetString("SMTPEmail"), viper.GetString("SMTPPassword"), viper.GetString("SMTPServer")), viper.GetString("SMTPEmail"), []string{recipent}, []byte(content))
		if e != nil {
			time.Sleep(5 * time.Second) // 5 seconds delay between retrys
		}
		return attempt < 3, e // try 3 times
	})
	if err != nil {
		return err
	}
	return nil
}
