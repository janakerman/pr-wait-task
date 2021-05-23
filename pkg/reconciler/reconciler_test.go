package reconciler_test

import (
	"context"
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/go-github/v35/github"
	"github.com/janakerman/pr-wait-task/pkg/reconciler"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	kreconciler "knative.dev/pkg/reconciler"
)

var _ = Describe("Reconciler", func() {
	var subject reconciler.Reconciler

	var (
		run      *v1alpha1.Run
		prGetter reconciler.PRGetter
		event    kreconciler.Event
	)

	BeforeEach(func() {
		run = &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					Kind:       reconciler.Kind,
					APIVersion: reconciler.APIVersion,
				},
				Params: []v1alpha1.Param{
					{Name: "pr-number", Value: v1alpha1.ArrayOrString{StringVal: "1"}},
					{Name: "repository", Value: v1alpha1.ArrayOrString{StringVal: "owner/repo"}},
				},
			},
		}
	})

	Context("fails to reconcile", func() {
		BeforeEach(func() {
			prGetter = func(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error) {
				return nil, nil, nil
			}
		})
		JustBeforeEach(func() {
			subject = reconciler.Reconciler{
				GetPR: prGetter,
			}
			event = subject.ReconcileKind(context.Background(), run)
		})

		When("name is present", func() {
			BeforeEach(func() {
				run.Spec.Ref.Name = "we don't want a name"
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("marks run as failed", func() {
				Expect(run.IsDone()).To(BeTrue())
				Expect(run.IsSuccessful()).To(BeFalse())
			})
		})

		When("no PR number is present", func() {
			BeforeEach(func() {
				run.Spec.Params = []v1alpha1.Param{
					{Name: "repository", Value: v1alpha1.ArrayOrString{StringVal: "owner/repo"}},
				}
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("marks the run as failed", func() {
				Expect(run.IsDone()).To(BeTrue())
				Expect(run.IsSuccessful()).To(BeFalse())
			})

			It("states the reason", func() {
				Expect(run.Status.Conditions).To(HaveLen(1))
				Expect(run.Status.Conditions[0].Message).To(Equal("pr-number param is required"))
			})
		})

		When("no repository parameter is present", func() {
			BeforeEach(func() {
				run.Spec.Params = []v1alpha1.Param{
					{Name: "pr-number", Value: v1alpha1.ArrayOrString{StringVal: "1"}},
				}
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("marks the run as failed", func() {
				Expect(run.IsDone()).To(BeTrue())
				Expect(run.IsSuccessful()).To(BeFalse())
			})

			It("states the reason", func() {
				Expect(run.Status.Conditions).To(HaveLen(1))
				Expect(run.Status.Conditions[0].Message).To(Equal("repository param is required"))
			})
		})

		When("repository parameter is invalid", func() {
			BeforeEach(func() {
				run.Spec.Params = []v1alpha1.Param{
					{Name: "pr-number", Value: v1alpha1.ArrayOrString{StringVal: "1"}},
					{Name: "repository", Value: v1alpha1.ArrayOrString{StringVal: "i-need-a-hyphen"}},
				}
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("marks the run as failed", func() {
				Expect(run.IsDone()).To(BeTrue())
				Expect(run.IsSuccessful()).To(BeFalse())
			})

			It("states the reason", func() {
				Expect(run.Status.Conditions).To(HaveLen(1))
				Expect(run.Status.Conditions[0].Message).To(Equal("unexpected repository format: i-need-a-hyphen"))
			})
		})

		When("unexpected parameter is present", func() {
			BeforeEach(func() {
				run.Spec.Params = []v1alpha1.Param{
					{Name: "pr-number", Value: v1alpha1.ArrayOrString{StringVal: "1"}},
					{Name: "repository", Value: v1alpha1.ArrayOrString{StringVal: "owner/repo"}},
					{Name: "not-wanted", Value: v1alpha1.ArrayOrString{StringVal: "anything"}},
				}
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("marks the run as failed", func() {
				Expect(run.IsDone()).To(BeTrue())
				Expect(run.IsSuccessful()).To(BeFalse())
			})

			It("states the reason", func() {
				Expect(run.Status.Conditions).To(HaveLen(1))
				Expect(run.Status.Conditions[0].Message).To(Equal("found unexpected params: [not-wanted]"))
			})
		})

		When("error occurs fetching PR", func() {
			BeforeEach(func() {
				prGetter = func(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error) {
					return nil, nil, errors.New("oh no")
				}
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("marks the run as failed", func() {
				Expect(run.IsDone()).To(BeTrue())
				Expect(run.IsSuccessful()).To(BeFalse())
			})

			It("states the reason", func() {
				Expect(run.Status.Conditions).To(HaveLen(1))
				Expect(run.Status.Conditions[0].Message).To(Equal("Failed to get PR: oh no"))
			})
		})
	})

	Context("reconciles successfully", func() {
		var (
			observedOwner, observedRepo string
			observedNumber              int
			state                       string
			createdTime                 time.Time
			merged                      bool
		)

		BeforeEach(func() {
			prGetter = func(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error) {
				observedOwner = owner
				observedRepo = repo
				observedNumber = number

				return &github.PullRequest{
					State:     &state,
					CreatedAt: &createdTime,
					Merged:    &merged,
				}, nil, nil
			}
		})

		JustBeforeEach(func() {
			subject = reconciler.Reconciler{
				GetPR: prGetter,
			}
			event = subject.ReconcileKind(context.Background(), run)
		})

		Context("run is not yet running", func() {
			When("PR is open", func() {
				BeforeEach(func() {
					state = "open"
					createdTime = time.Now()
				})
				It("calls GetPR with expected args", func() {
					Expect(observedOwner).To(Equal("owner"))
					Expect(observedRepo).To(Equal("repo"))
					Expect(observedNumber).To(Equal(1))
				})
				It("returns an event", func() {
					Expect(event).NotTo(BeNil())
				})
				It("sets start time", func() {
					Expect(run.Status.StartTime.Time).To(Equal(createdTime))
				})
				It("marks run as waiting", func() {
					Expect(run.Status.Conditions).To(HaveLen(1))
					Expect(run.Status.Conditions[0].Message).To(Equal("Waiting for PR to be merged"))
				})
			})
			When("PR is merged", func() {
				BeforeEach(func() {
					state = "closed"
					merged = true
					createdTime = time.Now()
				})
				It("calls GetPR with expected args", func() {
					Expect(observedOwner).To(Equal("owner"))
					Expect(observedRepo).To(Equal("repo"))
					Expect(observedNumber).To(Equal(1))
				})
				It("returns an event", func() {
					Expect(event).NotTo(BeNil())
				})
				It("sets start time", func() {
					Expect(run.Status.StartTime.Time).To(Equal(createdTime))
				})
				It("marks run as successful", func() {
					Expect(run.IsSuccessful()).To(BeTrue())
					Expect(run.Status.Conditions).To(HaveLen(1))
					Expect(run.Status.Conditions[0].Message).To(Equal("PR was merged"))
				})
			})
		})

		Context("run is running", func() {
			t := time.Now()

			BeforeEach(func() {
				startTime := metav1.NewTime(t)
				run.Status.StartTime = &startTime
			})

			When("PR is open", func() {
				BeforeEach(func() {
					state = "closed"
				})
				It("calls GetPR with expected args", func() {
					Expect(observedOwner).To(Equal("owner"))
					Expect(observedRepo).To(Equal("repo"))
					Expect(observedNumber).To(Equal(1))
				})
				It("returns an event", func() {
					Expect(event).NotTo(BeNil())
				})
				It("sets start time", func() {
					Expect(run.Status.StartTime.Time).To(Equal(t))
				})
				It("marks run as waiting", func() {
					Expect(run.Status.Conditions).To(HaveLen(1))
					Expect(run.Status.Conditions[0].Message).To(Equal("PR was merged"))
				})
			})
			When("PR is merged", func() {
				BeforeEach(func() {
					state = "closed"
				})
				It("calls GetPR with expected args", func() {
					Expect(observedOwner).To(Equal("owner"))
					Expect(observedRepo).To(Equal("repo"))
					Expect(observedNumber).To(Equal(1))
				})
				It("returns an event", func() {
					Expect(event).NotTo(BeNil())
				})
				It("sets start time", func() {
					Expect(run.Status.StartTime.Time).To(Equal(t))
				})
				It("marks run as waiting", func() {
					Expect(run.Status.Conditions).To(HaveLen(1))
					Expect(run.Status.Conditions[0].Message).To(Equal("PR was merged"))
				})
			})
		})

		When("Kind is unknown", func() {
			BeforeEach(func() {
				run.Spec.Ref.Kind = "NotInterested"
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("leaves the run unchanged", func() {
				Expect(run.IsDone()).To(BeFalse())
			})
		})

		When("APIVersion is unknown", func() {
			BeforeEach(func() {
				run.Spec.Ref.APIVersion = "NotInterested"
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("leaves the run unchanged", func() {
				Expect(run.IsDone()).To(BeFalse())
			})
		})

		When("run is finished", func() {
			BeforeEach(func() {
				run.Status.MarkRunSucceeded("done", "done")
			})

			It("returns no event", func() {
				Expect(event).To(BeNil())
			})

			It("leaves the run unchanged", func() {
				Expect(run.IsDone()).To(BeTrue())
			})
		})
	})
})
