package kubernetes

import rego.v1

good := {"kind": "Deployment", "metadata": {"name": "svc"}, "spec": {"template": {"spec": {
	"securityContext": {"runAsNonRoot": true},
	"containers": [{"name": "app", "image": "svc:1.0", "readinessProbe": {}, "resources": {"limits": {"memory": "64Mi"}},
		"securityContext": {"allowPrivilegeEscalation": false, "readOnlyRootFilesystem": true}}],
}}}}

test_compliant_deployment if {
	count(deny) == 0 with input as good
}

test_latest_and_root_denied if {
	bad := json.patch(good, [
		{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "svc:latest"},
		{"op": "replace", "path": "/spec/template/spec/securityContext/runAsNonRoot", "value": false},
	])
	count(deny) == 2 with input as bad
}
