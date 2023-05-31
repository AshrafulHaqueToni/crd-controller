#!/bin/bash
set -x
vendor/k8s.io/code-generator/generate-groups.sh all \
  github.com/AshrafulHaqueToni/crd-controller/pkg/client \
  github.com/AshrafulHaqueToni/crd-controller/pkg/apis \
  ash.dev:v1alpha1 \
  --go-header-file /home/ashraful/go/src/github.com/AshrafulHaqueToni/crd-controller/hack/boilerplate.go.txt
#
controller-gen rbac:roleName=controller-perms crd paths=github.com/AshrafulHaqueToni/crd-controller/pkg/apis/ash.dev/v1alpha1 \
 crd:crdVersions=v1 output:crd:dir=/home/ashraful/go/src/github.com/AshrafulHaqueToni/crd-controller/manifests output:stdout