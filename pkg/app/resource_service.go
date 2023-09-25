/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package app

import (
	"github.com/codeallergy/glue"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	htmlTemplate "html/template"
	"io/ioutil"
	"strings"
	"sync"
	textTemplate "text/template"
)

type implResourceService struct {

	Context              glue.Context    `inject`
	ResourceSources      []*glue.ResourceSource  `inject:"optional"`

	textTemplates sync.Map
	htmlTemplates sync.Map
}

func ResourceService() sprint.ResourceService {
	return &implResourceService{}
}

func (t *implResourceService) GetResource(name string) (content []byte, err error) {

	defer sprintutils.PanicToError(&err)

	res, ok := t.Context.Resource(name)
	if !ok {
		return nil, errors.Errorf("resource not found '%s'", name)
	}

	file, err := res.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ioutil.ReadAll(file)
}

func (t *implResourceService) TextTemplate(fileName string) (tmpl *textTemplate.Template, err error) {

	if val, ok := t.textTemplates.Load(fileName); ok {
		if tpl, ok := val.(*textTemplate.Template); ok {
			return tpl, nil
		}
	}
	res, err := t.GetResource(fileName)
	if err != nil {
		return nil, err
	}
	tpl, err := textTemplate.New(fileName).Parse(string(res))
	if err != nil {
		return nil, err
	}
	t.textTemplates.Store(fileName, tpl)
	return tpl, nil
}

func (t *implResourceService) HtmlTemplate(fileName string) (tmpl *htmlTemplate.Template, err error) {

	if val, ok := t.htmlTemplates.Load(fileName); ok {
		if tpl, ok := val.(*htmlTemplate.Template); ok {
			return tpl, nil
		}
	}
	res, err := t.GetResource(fileName)
	if err != nil {
		return nil, err
	}
	tpl, err := htmlTemplate.New(fileName).Parse(string(res))
	if err != nil {
		return nil, err
	}
	t.htmlTemplates.Store(fileName, tpl)
	return tpl, nil
}

func (t *implResourceService) GetLicenses(name string) (output string, err error) {

	content, err := t.GetResource(name)
	if err != nil {
		return "", err
	}

	packageName := t.Context.Properties().GetString("application.package", "")
	if packageName != "" {
		return filterLines(string(content), packageName), nil
	}

	return string(content), nil
}

func (t *implResourceService) GetOpenAPI(source string) string {
	var out strings.Builder

	for _, resourceSource := range t.ResourceSources {
		if resourceSource.Name != source {
			continue
		}
		for _, name := range resourceSource.AssetNames {
			if strings.HasSuffix(name, ".swagger.json") {
				if content, err := t.GetResource(name); err == nil {
					out.WriteString(string(content))
				}
			}
		}
	}

	return out.String()
}

func filterLines(content string, words ...string) string {

	var out strings.Builder

	for _, line := range strings.Split(content, "\n") {
		include := true
		for _, word := range words {
			if strings.Contains(line, word) {
				include = false
				break
			}
		}
		if include {
			out.WriteString(line)
			out.WriteRune('\n')
		}
	}

	return out.String()
}
