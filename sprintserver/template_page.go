/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintserver

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"html/template"
	"net/http"
)

type implTemplatePage struct {
	glue.InitializingBean

	pattern      string
	templateFile string
	tpl          *template.Template

	ResourceService sprint.ResourceService `inject`
}

func TemplatePage(pattern, templateFile string) sprint.Page {
	return &implTemplatePage{
		pattern: pattern,
		templateFile: templateFile,
	}
}

func (t *implTemplatePage) PostConstruct() (err error) {
	t.tpl, err = t.ResourceService.HtmlTemplate(t.templateFile)
	if err != nil {
		return errors.Errorf("template index file '%s' error, %v", t.templateFile, err)
	}
	return
}

func (t *implTemplatePage) Pattern() string {
	return t.pattern
}

func (t *implTemplatePage) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if r := recover(); r != nil {
			http.Error(w, fmt.Sprintf("%v", r), http.StatusInternalServerError)
		}
	}()

	r.ParseForm()
	t.tpl.Execute(w, r)
}
