![lint](https://github.com/fabiendupont/habana-ai-operator/actions/workflows/lint.yaml/badge.svg)
![tests](https://github.com/fabiendupont/habana-ai-operator/actions/workflows/test.yaml/badge.svg)
[![codecov](https://codecov.io/gh/fabiendupont/habana-ai-operator/branch/main/graph/badge.svg?token=EMH9QLP6NR)](https://codecov.io/gh/fabiendupont/habana-ai-operator)
[![go report](https://goreportcard.com/badge/github.com/fabiendupont/habana-ai-operator)](https://goreportcard.com/report/github.com/fabiendupont/habana-ai-operator)
![Build and push images](https://github.com/fabiendupont/habana-ai-operator/actions/workflows/images.yaml/badge.svg)

# Habana AI Operator

## TL;DR

```bash
# Deploy the Kernel Module Management Operator
$ kubectl apply -k https://github.com/kubernetes-sigs/kernel-module-management/config/default

# Deploy the Habana AI Operator
$ git clone git@github.com:fabiendupont/habana-ai-operator.git && cd habana-ai-operator
$ make deploy

# Create a sample DeviceConfig that targets all DL1 nodes.
$ kubectl apply -f config/samples/habana.ai_v1alpha1_deviceconfig.yaml

# Wait until all Habana AI components are healthy
$ kubectl get -n habana-ai-operator get all

# Run a sample workload pod
$ cat <<EOF kubectl -f -
apiVersion: v1
kind: Pod
metadata:
  name: hl-smi
  namespace: habana-ai-operator
spec:
  restartPolicy: OnFailure
  containers:
  - image: ghcr.io/fabiendupont/habana-ai-hl-smi:latest
    imagePullPolicy: IfNotPresent
    name: hl-smi
    resources:
      limits:
        habana.ai/gaudi: "1"
    securityContext:
      capabilities:
        add:
          - "SYS_RAWIO"
EOF

$ kubectl logs -n default pod/hl-smi
```

## Overview

Kubernetes provides access to special hardware resources such as Habana AI accelerators, NICs,
and other devices through the [device plugin framework](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/).

However, configuring and managing Kubernetes nodes with these hardware resources requires
configuration of multiple software components, such as drivers, container runtimes, device plugins
and other libraries, which is time consuming and error prone.

The Habana AI [Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) is used in
paar with the [Kernel Module Management Operator](https://github.com/kubernetes-sigs/kernel-module-management), in order to automate
the management of all the Habana AI software components needed to provision AI accelerators within Kubernetes, from
the drivers to the respective monitoring metrics.

## Dependencies

- [Kernel Module Management Operator](https://github.com/kubernetes-sigs/kernel-module-management)
- [Node Feature Discovery](https://github.com/kubernetes-sigs/node-feature-discovery)

## Components

The components managed by the operator are:

- [Kernel Module Management Operator Module](https://github.com/kubernetes-sigs/kernel-module-management/blob/main/api/v1beta1/module_types.go), which
  manages:
  - [Habana Labs drivers](https://github.com/fabiendupont/habana-ai-driver)
  - [Kubernetes device plugin for Habana AI](https://docs.habana.ai/en/latest/Orchestration/Gaudi_Kubernetes/Habana_Device_Plugin_for_Kubernetes.html)
- [Habana AI Prometheus Metrics Exporter](https://docs.habana.ai/en/latest/Orchestration/Gaudi_Kubernetes/Prometheus_Metric_Exporter_for_Kubernetes.html)

## Design

For a detailed description of the design of the Habana AI Operator check this [doc](./docs/design.md).

## OpenShift

Use the Habana AI Operator, along with the [Kernel Module Management Operator](https://github.com/kubernetes-sigs/kernel-module-management)
in your [OpenShift](https://www.redhat.com/en/technologies/cloud-computing/openshift) cluster to
automatically provision and manage different AI accelerators configurations per node group.

The following guide leverages the automatically generated container images of:
- the Habana AI Operator
- the operator [OLM bundle](https://operator-framework.github.io/olm-book/docs/glossary.html#bundle)
- the [OLM CatalogSource](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/architecture.md)

```bash
# Given an OpenShift cluster with Habana AI powered nodes
$ oc get clusterversions.config.openshift.io
NAME      VERSION   AVAILABLE   PROGRESSING   SINCE   STATUS
version   4.11.0    True        False         30h     Cluster version is 4.11.0

# Enable alternative firmware path on worker nodes
$ oc apply -f https://github.com/fabiendupont/habana-ai-operator/hack/openshift/machineconfig-firmware-path.yaml

# Deploy the Kernel Module Management Operator
$ oc apply -k https://github.com/kubernetes-sigs/kernel-module-management/config/default

# Deploy the Habana AI Operator via OLM
$ oc apply -f https://github.com/fabiendupont/habana-ai-operator/hack/openshift/deploy.yaml

# Create a sample DeviceConfig targeting Habana AI nodes.
$ oc apply -f https://github.com/fabiendupont/habana-ai-operator/hack/openshift/deviceconfig.yaml

# Wait for all Habana AI components to be healthy
$ oc get -n habana-ai-operator all

# Verify the setup by running a sample workload pod
$ oc apply -f https://github.com/fabiendupont/habana-ai-operator/hack/openshift/sample-workload.yaml

# Check the workload logs
$ oc logs -n default pod/hl-smi
```
