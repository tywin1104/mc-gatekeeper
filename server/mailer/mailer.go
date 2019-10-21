package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	c "github.com/tywin1104/mc-whitelist/config"
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
func Send(templateName string, templateData interface{}, subject string, recipent string, appConfig *c.Config) error {
	body, err := parseTemplate(templateName, templateData)
	if err != nil {
		return err
	}
	content := "To: " + recipent + "\r\nSubject: " + subject + "\r\n" + mime + "\r\n" + body
	SMTP := fmt.Sprintf("%s:%d", "smtp.mailgun.org", appConfig.SMTPPort)
	err = smtp.SendMail(SMTP, smtp.PlainAuth("", appConfig.SMTPEmail, appConfig.SMTPPassword, appConfig.SMTPServer), appConfig.SMTPEmail, []string{recipent}, []byte(content))
	return err
}
