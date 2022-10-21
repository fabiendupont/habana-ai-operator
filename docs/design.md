# Habana AI Operator

## References

- Habana Gaudi [docs](https://docs.habana.ai/en/latest/)
- Kernel Module Management (KMM) operator (KMMO) [repository](https://github.com/kubernetes-sigs/kernel-module-management)

## Summary

The Habana AI Operator fullfils the goal to seamlessly enable Habana AI accelerators on Kubernetes and OpenShift.
It provides an opinionated and extendable API. It offloads the driver container and the device plugin to
[KMMO](https://github.com/kubernetes-sigs/kernel-module-management/blob/design/docs/design/fundamentals.md).
It follows the software engineering practices, leading to great maintainability, reliability and development velocity.

### User Stories

#### User Story 1

As a user I want to enable the Habana AI hardware accelerators on Kubernetes/OpenShift on a group of nodes of my
choice, with the driver version and configuration of my choice. But I want to have as minimum configuration options as possible.

## Design Details

The following sections describe the design decisions and trade-offs of each implementation detail of the Habana AI Operator.

### API

The Habana AI Operator starts by designing an extremely lean API that trades configuration flexibility with reliability,
as the operator takes ownership of setting up and lifecycling all required components with minimum user input.

This trade-off leads to:

- easy and robust dependency management
- simple API
- seamless user experience
- easier extendability
- small and focused codebase, as all flows are highly opinionated

#### DeviceConfig

The `DeviceConfig` is the main [Custom Resource Definition](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/) (CRD) of the Habana AI Operator.

##### DeviceConfigSpec

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| DriverImage | The Habana Labs driver image to use | string | true |
| DriverVersion | The Habana Labs Driver version to use | string | true |
| NodeSelector | Specifies the node selector to be used for this DeviceConfig | map[string]string |false |

The `DeviceConfig` specification has the following goals:

- support multiple `DeviceConfig`s on a cluster, each one targeting a unique group of nodes via a
  [NodeSelector](https://pkg.go.dev/k8s.io/api/core/v1#PodSpec)
- each `DeviceConfig` can have a different driver configuration
- the `DeviceConfig` should accept minimum user input

![DeviceConfig Example](./assets/deviceconfig-example.png)

### Node Selector Validation

The Habana AI Operator supports multiple `DeviceConfig`s with different driver configurations on
the same cluster. This is implemented by including a node selector in its specification. But a node
can only have one driver configuration, as it cannot have more than one kernel modules owning the
same device. The operator therefore, needs to validate each `DeviceConfig` applied by the user, in
order to verify that its node selector does not include a node that is already part of another
`DeviceConfig` node selector.

To validate the uniqueness of node selectors among the `DeviceConfig`s, when a new `DeviceConfig` is
created, the following validation is performed:

![DeviceConfig Validation Flowchart](./assets/deviceconfig-nodeselector-validation-flowchart.png)

### Kernel Module Management (KMM) Operator Integration

The Habana AI Operator integrates with [KMM](https://github.com/kubernetes-sigs/kernel-module-management) to offload the
management of the Habana Labs driver container and the Habana Labs device plugin on a Kubernetes or
OpenShift cluser. This integration helps the Habana AI Operator focus on its user experience and
features, while gaining from the [KMM features](https://github.com/kubernetes-sigs/kernel-module-management/blob/design/docs/design/fundamentals.md)
and reducing its own codebase.

![KMM Operator Integration](./assets/kmm-operator-integration.png)

### Unit Testing

The current test coverage is above `70%`, with the most critical parts of the operator already
covered. The frameworks and mocking tools adopted are described below.

#### Frameworks

The Habana AI Operator unit tests are written in:
- [Ginkgo](https://pkg.go.dev/github.com/onsi/ginkgo/v2)
- [gomega](https://pkg.go.dev/github.com/onsi/gomega)

Their integration in [kubebuilder](https://book.kubebuilder.io/cronjob-tutorial/writing-tests.html), huge
adoption in the Kubernetes operator ecosystem and active development, make them a robust choice.

#### Mocks

Mocking internal and external packages increases the [testability](https://en.wikipedia.org/wiki/Software_testability)
of the operator. In Go one should not look further than:

- [gomock](https://pkg.go.dev/github.com/golang/mock/gomock)
- [mockgen](https://pkg.go.dev/github.com/golang/mock/mockgen)

Using [go generate](https://pkg.go.dev/cmd/go/internal/generate) and [mockgen](https://pkg.go.dev/github.com/golang/mock/mockgen)
all mocks can be automatically generated.

### Linting

[Linting](https://en.wikipedia.org/wiki/Lint_(software)) helps to not accumulate technical debt and
keep a consistent codebase. The Habana AI Operator leverages [golangci-lint](https://github.com/golangci/golangci-lint)
with its [default linters](https://golangci-lint.run/usage/linters/#enabled-by-default) enabled.

## Future Work

### Conditions

`DeviceConfigStatus` conditions and thrive to adhere to the
[respective suggestions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
of the Kubernetes community. There are currently 2 conditions:

- `Ready`
- `Errored`

But while these 2 conditions are all we need, they currently only track the result of the creation
or patching of the managed CRs and not their actual status. As a future enhancement, the
`DeviceConfig` controller will watch changes in the status of the managed CRs, e.g. the KMM
`Module`, in order to update the `DeviceConfig`'s conditions with the respective [reasons](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition).
