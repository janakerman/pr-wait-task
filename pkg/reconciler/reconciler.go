package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	APIVersion = "github.com/janakerman/v0"
	Kind       = "PRWait"
	PRParam    = "pr-number"
)

type Reconciler struct {
	EnqueueAfter func(interface{}, time.Duration)
}

type params struct {
	prNum string
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

	_ = params

	return kreconciler.NewEvent(corev1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", r.Namespace, r.Name)
}

func getParams(r *v1alpha1.Run) (*params, error) {
	expr := r.Spec.GetParam(PRParam)
	if expr == nil || expr.Value.StringVal == "" {
		return nil, errors.New("pr-number param is required")
	}
	if len(r.Spec.Params) != 1 {
		var found []string
		for _, p := range r.Spec.Params {
			if p.Name == PRParam {
				continue
			}
			found = append(found, p.Name)
		}

		return nil, fmt.Errorf("found unexpected params: %v", found)
	}
	return &params{
		prNum: expr.Value.StringVal,
	}, nil
}
