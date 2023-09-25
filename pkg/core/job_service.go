/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"go.uber.org/zap"
	"strings"
	"sync"
)

var ErrJobNotFound = errors.New("job not found")

type implJobService struct {
	Log           *zap.Logger              `inject`

	muJobs  sync.Mutex
	jobs    []*sprint.JobInfo
}

func JobService() sprint.JobService {
	return &implJobService{}
}

func (t *implJobService) ListJobs() ([]string, error) {
	t.muJobs.Lock()
	defer t.muJobs.Unlock()

	var list []string
	for _, job := range t.jobs {
		list = append(list, job.Name)
	}

	return list, nil
}

func (t *implJobService) AddJob(job *sprint.JobInfo) error {
	t.muJobs.Lock()
	defer t.muJobs.Unlock()

	t.jobs = append(t.jobs, job)
	return nil
}

func (t *implJobService) CancelJob(name string) error {
	t.muJobs.Lock()
	defer t.muJobs.Unlock()

	for i, job := range t.jobs {
		if job.Name == name {
			t.jobs = append(t.jobs[:i], t.jobs[i+1:]...)
			return nil
		}
	}

	return ErrJobNotFound
}

func (t *implJobService) RunJob(ctx context.Context, name string) (err error) {

	defer sprintutils.PanicToError(&err)

	job, err := t.findJob(name)
	if err != nil {
		return err
	}

	return job.ExecutionFn(ctx)
}

func (t *implJobService) findJob(name string) (*sprint.JobInfo, error) {
	t.muJobs.Lock()
	defer t.muJobs.Unlock()

	for _, job := range t.jobs {
		if job.Name == name {
			return job, nil
		}
	}

	return nil, ErrJobNotFound
}


func (t *implJobService) ExecuteCommand(cmd string, args []string) (string, error) {

	switch cmd {
	case "list":
		list, err := t.ListJobs()
		if err != nil {
			return "", err
		}
		return strings.Join(list, "\n"), nil

	case "run":
		if len(args) < 1 {
			return "Usage: job run name", nil
		}
		jobName := args[0]
		go func() {

			err := t.RunJob(context.Background(), jobName)
			if err != nil {
				t.Log.Error("JobRun", zap.String("jobName", jobName), zap.Error(err))
			}

		}()
		return "OK", nil

	case "cancel":
		if len(args) < 1 {
			return "Usage: job cancel name", nil
		}
		jobName := args[0]
		err := t.CancelJob(jobName)
		if err != nil {
			return "", errors.Errorf("cancel of job '%s' was failed, %v", jobName, err)
		}
		return"OK", nil

	default:
		return "", errors.Errorf("unknown job command '%s'", cmd)
	}

}