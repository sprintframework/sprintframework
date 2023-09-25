/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcore

import (
	"bytes"
	"context"
	"errors"
	"github.com/mailgun/mailgun-go/v4"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"go.uber.org/zap"
	"strings"
	"time"
)

type implMailService struct {
	Properties      glue.Properties     `inject`
	ResourceService sprint.ResourceService `inject`
	Log             *zap.Logger           `inject`
}

func MailService() sprint.MailService {
	return &implMailService{}
}

func (t *implMailService) BeanName() string {
	return "mail_service"
}

func (t *implMailService) SendMail(mail *sprint.Mail, timeout time.Duration, async bool) error {

	key := t.Properties.GetString("mailgun.key", "")

	if key == "" {
		return errors.New("empty property 'mailgun.key'")
	}

	tmpl, err := t.ResourceService.TextTemplate(mail.TextTemplate)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	err = tmpl.Execute(&body, mail.Data)
	if err != nil {
		return err
	}

	mg := mailgun.NewMailgun(t.getDomainFromEmail(mail.Sender), key)

	message := mg.NewMessage(mail.Sender, mail.Subject, body.String(), mail.Recipients...)

	if mail.HtmlTemplate != "" {

		htmlTmpl, err := t.ResourceService.HtmlTemplate(mail.HtmlTemplate)
		if err != nil {
			return err
		}

		var htmlBody bytes.Buffer
		err = htmlTmpl.Execute(&htmlBody, mail.Data)
		if err != nil {
			return err
		}

		message.SetHtml(htmlBody.String())
	}

	for _, attachment := range mail.Attachments {

		content, err := t.ResourceService.GetResource(attachment)
		if err != nil {
			return err
		}

		message.AddBufferAttachment(attachment, content)

	}

	sendFn := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		resp, id, err := mg.Send(ctx, message)

		if err == nil {
			t.Log.Info("SendMail", zap.String("id", id), zap.String("resp", resp))
		} else {
			t.Log.Error("SendMail", zap.String("sender", mail.Sender), zap.Strings("recipient", mail.Recipients), zap.Error(err))
		}
		return err
	}

	if async {
		go sendFn()
		return nil
	} else {
		return sendFn()
	}

}

func  (t *implMailService) getDomainFromEmail(email string) string {

	i := strings.LastIndex(email, "@")
	if i == -1 {
		return email
	}

	return email[i+1:]
}
