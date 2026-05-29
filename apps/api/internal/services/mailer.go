package services

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"mime"
	"net"
	"net/smtp"
	"strings"

	"school-oj/apps/api/internal/config"
)

type Mailer struct {
	Cfg config.Config
}

func GenerateSixDigitCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (m Mailer) SendVerificationCode(to, purpose, code string) error {
	subject := "黄海在线验证码"
	body := fmt.Sprintf("您的黄海在线验证码是 %s，10 分钟内有效。", code)
	if purpose == "password_reset" {
		body = fmt.Sprintf("您正在找回黄海在线账号密码，验证码是 %s，10 分钟内有效。", code)
	}
	if purpose == "rebind_email" {
		body = fmt.Sprintf("您正在换绑黄海在线账号邮箱，验证码是 %s，10 分钟内有效。", code)
	}
	return m.Send(to, subject, body)
}

func (m Mailer) Send(to, subject, body string) error {
	if strings.TrimSpace(m.Cfg.SMTPHost) == "" {
		log.Printf("mail disabled: to=%s subject=%s body=%s", to, subject, body)
		return nil
	}
	addr := net.JoinHostPort(m.Cfg.SMTPHost, fmt.Sprintf("%d", m.Cfg.SMTPPort))
	fromName := mime.QEncoding.Encode("utf-8", m.Cfg.MailFromName)
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s <%s>\r\n", fromName, m.Cfg.MailFrom)
	fmt.Fprintf(&msg, "To: <%s>\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprint(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprint(&msg, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprint(&msg, "\r\n")
	fmt.Fprint(&msg, body)
	var auth smtp.Auth
	if m.Cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", m.Cfg.SMTPUsername, m.Cfg.SMTPPassword, m.Cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, m.Cfg.MailFrom, []string{to}, msg.Bytes())
}
