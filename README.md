# Custom Resource Definition and Sample Controller #
## Goals ##
- Understand CRs
- Define a CR
- Understand Controller
- Write a sample Controller

## Code Gen ##
Code-gen generating:
- DeepCopyObject
- Clientset
- Informer
- Lister

## Info ##
- Group Name: ash.dev
- Version Name: v1alpha1
- Resource Name: Ash

## Procedure ##

- import `"k8s.io/code-generator"` into `main.go`
- run `go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest`
- run `go mod tidy;go mod vendor`
- run `chmod +x /hack/codegen.sh`
- run `chmod +x vendor/k8s.io/code-generator/generate-groups.sh`
- run `hack/codegen.sh`
- again run `go mod tidy;go mod vendor`

## Deploy Custom resources ##
- Just create a yaml file like `manifests/ash.yaml`.
- Run `kubectl apply -f manifests/ash.yaml`.
- Run `kubectl get ash`.

## Resource ##
https://www.linkedin.com/pulse/kubernetes-custom-controllers-part-1-kritik-sachdeva/

https://www.linkedin.com/pulse/kubernetes-custom-controller-part-2-kritik-sachdeva/

https://github.com/ishtiaqhimel/crd-controller/tree/master