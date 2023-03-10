# ocf-operator

1. kubebuilder init --domain iot.dev --repo=github.com/vitu1234/ocf-operator
2. kubebuilder create api --group iot --version v1alpha1 --kind OCFDevice
3. Make manifests - generates CRs and CRDs
4. Make install - runs the controller against our cluster. 
5. 