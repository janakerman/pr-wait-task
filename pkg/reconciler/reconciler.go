package reconciler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	APIVersion = "github.com/janakerman/v0"
	Kind       = "PRWait"
	ParamPR    = "pr-number"
	ParamRepo  = "repository"
)

type PRGetter func(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error)

type params struct {
	prNum       int
	owner, repo string
}

type Reconciler struct {
	EnqueueAfter func(interface{}, time.Duration)
	GetPR        PRGetter
}

func NewRconciler(client *github.Client, enqueAfter func(interface{}, time.Duration)) Reconciler {
	return Reconciler{
		EnqueueAfter: enqueAfter,
		GetPR: func(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error) {
			return client.PullRequests.Get(ctx, owner, repo, number)
		},
	}
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, r *v1alpha1.Run) kreconciler.Event {
	if r.Spec.Ref == nil ||
		r.Spec.Ref.APIVersion != APIVersion || r.Spec.Ref.Kind != Kind {
		// This is not a Run we should have been notified about; do nothing.
		return nil
	}

	if r.Spec.Ref.Name != "" {
		r.Status.MarkRunFailed("UnexpectedName", "Found unexpected ref name: %s", r.Spec.Ref.Name)
		return nil
	}

	// Ignore completed waits.
	if r.IsDone() {
		log.Printf("Run is finished, done reconciling\n")
		return nil
	}

	params, err := getParams(r)
	if err != nil {
		r.Status.MarkRunFailed("UnexpectedParams", err.Error())
		return nil
	}

	pr, _, err := c.GetPR(ctx, params.owner, params.repo, params.prNum)
	if err != nil {
		r.Status.MarkRunFailed("GithubError", "Failed to get PR: %s", err)
		return nil
	}

	if r.Status.StartTime == nil {
		now := metav1.NewTime(pr.GetCreatedAt())
		r.Status.StartTime = &now
	}

	if pr.GetState() == "closed" {
		if pr.GetMerged() {
			r.Status.MarkRunSucceeded("Merged", "PR was merged")
		} else {
			r.Status.MarkRunFailed("NotMerged", "PR was closed without merging")
		}
	} else {
		r.Status.MarkRunRunning("Waiting", "Waiting for PR to be merged")
	}

	return kreconciler.NewEvent(corev1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", r.Namespace, r.Name)
}

func getParams(r *v1alpha1.Run) (*params, error) {
	prNumStr, err := getParam(r, ParamPR)
	if err != nil {
		return nil, err
	}
	prNum, err := strconv.Atoi(prNumStr)
	if err != nil {
		return nil, fmt.Errorf("pr-number not a number: %s", prNumStr)
	}

	repo, err := getParam(r, ParamRepo)
	if err != nil {
		return nil, err
	}
	split := strings.Split(repo, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("unexpected repository format: %s", repo)
	}

	if len(r.Spec.Params) != 1 {
		var found []string
		for _, p := range r.Spec.Params {
			if p.Name == ParamPR || p.Name == ParamRepo {
				continue
			}
			found = append(found, p.Name)
		}
		if len(found) > 0 {
			return nil, fmt.Errorf("found unexpected params: %v", found)
		}
	}
	return &params{
		prNum: prNum,
		owner: split[0],
		repo:  split[1],
	}, nil
}

func getParam(r *v1alpha1.Run, param string) (string, error) {
	expr := r.Spec.GetParam(param)
	if expr == nil || expr.Value.StringVal == "" {
		return "", fmt.Errorf("%s param is required", param)
	}
	return expr.Value.StringVal, nil
}
