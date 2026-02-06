package mailer

import (
	"bytes"
	"embed"
	ht "html/template"
	tt "text/template"
	"time"

	"github.com/wneessen/go-mail"
)

//go:embed "templates"
var templateFS embed.FS

// Mailer содержит экземпляр струкруры для соединения с SMTP-сервером и строку отпраитель "ФИО"
type Mailer struct {
	client *mail.Client
	sender string
}

// New - вернет экземпляр Mailer содержащий инфо о клиенте и отправителе.
func New(host string, port int, username, password, sender string) (*Mailer, error) {
	//инициализируем экземпояр соотв-ми настройками
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

	mailer := &Mailer{
		client: client,
		sender: sender,
	}

	return mailer, nil
}

// Send - принимает адрес электронной почты(1ый пар-р), имя файла содерж-го шаблоны, и динамические данные для шаблона.
func (m *Mailer) Send(recipient string, templateFile string, data any) error {
	// парсим файл шаблона из встроенной файловой системы.
	textTmpl, err := tt.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	//выполнить шаблон subject
	subject := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	//выполнить шаблон plainBody
	plainBody := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}
	// используйте ParseFS html/template для парсинга файла шаблона из встроенной ФС
	htmlTmpl, err := ht.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}
	//выполнить шаблон  htmlBody
	htmlBody := new(bytes.Buffer)
	err = htmlTmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Используйте ф-ию инициализации нового экземпляра mail.NewMSG.
	msg := mail.NewMsg()

	//установите получателя
	err = msg.To(recipient)
	if err != nil {
		return err
	}

	//установите отправителя
	err = msg.From(m.sender)
	if err != nil {
		return err
	}

	// установка тела сообщения в обычном формате
	msg.Subject(subject.String())
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())
	// и установка html-body
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())

	// установит соедиение с SMTP сервером, отправит сообщение, закроет соединение.
	// пробуем отправить письмо 3 раза, с задержками. После чего ->err
	for i := 1; i <= 3; i++ {
		err = m.client.DialAndSend(msg)
		if err == nil {
			return nil
		}
		if i != 3 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return err
}
