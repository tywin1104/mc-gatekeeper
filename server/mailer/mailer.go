package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

const (
	mime = "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
)

var config = &smtpConfig{}

func init() {
	config.load()
}

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
	SMTP := fmt.Sprintf("%s:%d", "smtp.mailgun.org", config.Port)
	err = smtp.SendMail(SMTP, smtp.PlainAuth("", config.Email, config.Password, config.Server), config.Email, []string{recipent}, []byte(content))
	return err
}
