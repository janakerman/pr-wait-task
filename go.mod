module github.com/janakerman/pr-wait-task

go 1.16

require (
	github.com/google/go-github/v35 v35.2.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/tektoncd/pipeline v0.20.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	knative.dev/pkg v0.0.0-20210119162123-1bbf0a6436c3
)

replace k8s.io/client-go => k8s.io/client-go v0.20.2
