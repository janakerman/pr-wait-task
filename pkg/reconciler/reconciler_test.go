package reconciler_test

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/janakerman/pr-wait-task/pkg/reconciler"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

var _ = Describe("Reconciler", func() {
	var subject reconciler.Reconciler

	BeforeSuite(func() {
		subject = reconciler.Reconciler{}
	})

	Context("reconciles unknown kind", func() {
		run := &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					Kind:       "NotInterested",
					APIVersion: "not.my.api.version",
				},
				Params: []v1alpha1.Param{
					{Name: "Number", Value: v1beta1.ArrayOrString{StringVal: "1"}},
				},
			},
		}
		err := subject.ReconcileKind(context.Background(), run)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

		It("leaves the run unchanged", func() {
			Expect(run.IsDone()).To(BeFalse())
		})
	})

	Context("reconciles with name", func() {
		run := &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					Kind:       reconciler.Kind,
					APIVersion: reconciler.APIVersion,
					Name:       "we don't want a name",
				},
				Params: []v1alpha1.Param{
					{Name: "Number", Value: v1beta1.ArrayOrString{StringVal: "1"}},
				},
			},
		}
		err := subject.ReconcileKind(context.Background(), run)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

		It("marks run as failed", func() {
			Expect(run.IsDone()).To(BeTrue())
			Expect(run.IsSuccessful()).To(BeFalse())
		})
	})

	Context("reconciles completed run", func() {
		run := &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					Kind:       reconciler.Kind,
					APIVersion: reconciler.APIVersion,
					Name:       "we don't want a name",
				},
				Params: []v1alpha1.Param{
					{Name: "Number", Value: v1beta1.ArrayOrString{StringVal: "1"}},
				},
			},
		}
		run.Status.MarkRunSucceeded("done", "done")
		err := subject.ReconcileKind(context.Background(), run)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

		It("leaves the run unchanged", func() {
			Expect(run.IsDone()).To(BeTrue())
			Expect(run.IsSuccessful()).To(BeFalse())
		})
	})

	Context("reconciles without PR parameter", func() {
		run := &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					Kind:       reconciler.Kind,
					APIVersion: reconciler.APIVersion,
				},
			},
		}
		err := subject.ReconcileKind(context.Background(), run)

		It("does not error", func() {
			Expect(err).To(BeNil())
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

	Context("reconciles with unknown parameter", func() {
		run := &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					Kind:       reconciler.Kind,
					APIVersion: reconciler.APIVersion,
				},
				Params: []v1alpha1.Param{
					{Name: "not-wanted", Value: v1alpha1.ArrayOrString{StringVal: "anything"}},
					{Name: "pr-number", Value: v1alpha1.ArrayOrString{StringVal: "anything"}},
				},
			},
		}
		err := subject.ReconcileKind(context.Background(), run)

		It("does not error", func() {
			Expect(err).To(BeNil())
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
})
