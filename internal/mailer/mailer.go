package mailer

import (
	"bytes"
	"embed"
	"time"

	"github.com/wneessen/go-mail"

	// these packages need different aliases to be disambiguated due to "template"
	// BEK Note: im not a real fan of these names but its a minor issue so ill leave it.
	ht "html/template"
	tt "text/template"
)

// directive below embeds the templates into the bin
// i dont agree with idea of "directives" but it works
// i'd prefer it if it was a function that had an error or some object with a method.

//go:embed "templates"
var templateFS embed.FS

// mailer struct contains the mail client instance and sender info
type Mailer struct {
	client *mail.Client
	sender string
}

// mailer constructor
func New(host string, port int, username, password, sender string) (*Mailer, error) {
	// init client
	client, err := mail.NewClient(
		host,
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithPort(port),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTimeout(5*time.Second),
	)

	if err != nil {
		return nil, err
	}

	// setup instance
	mailer := &Mailer{
		client: client,
		sender: sender,
	}

	return mailer, nil
}

// uses a file to send mail the recipient
func (m *Mailer) Send(recipient string, templateFile string, data any) error {

	// use parseFS to parse the template file from the embedded FS
	textTmpl, err := tt.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// execute "subject" template, passing in dynamic data
	subject := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	// do the same for the "plainBody" template
	plainBody := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	// parse required html files
	htmlTmpl, err := ht.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// execute template (build view)
	htmlBody := new(bytes.Buffer)
	err = htmlTmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// init message
	msg := mail.NewMsg()
	// add recipient
	err = msg.To(recipient)
	if err != nil {
		return err
	}
	// add sender
	err = msg.From(m.sender)
	if err != nil {
		return err
	}
	// set subject, body, alt
	msg.Subject(subject.String())
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())

	// open server, send message, close server
	return m.client.DialAndSend(msg)
}
