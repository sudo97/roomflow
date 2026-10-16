# Golden path: вимоги до маніфестів Kubernetes (ресурси, проби, безпека, версії образів).
package kubernetes

import rego.v1

workloads contains w if {
	input.kind in {"Deployment", "StatefulSet", "DaemonSet"}
	w := input
}

containers contains c if {
	some w in workloads
	some c in w.spec.template.spec.containers
}

deny contains msg if {
	some c in containers
	not c.resources.limits.memory
	msg := sprintf("%s/%s: потрібен ліміт пам'яті", [input.metadata.name, c.name])
}

deny contains msg if {
	some c in containers
	not c.readinessProbe
	msg := sprintf("%s/%s: потрібна readinessProbe", [input.metadata.name, c.name])
}

deny contains msg if {
	some c in containers
	endswith(c.image, ":latest")
	msg := sprintf("%s/%s: тег latest заборонено", [input.metadata.name, c.name])
}

deny contains msg if {
	some w in workloads
	not w.spec.template.spec.securityContext.runAsNonRoot
	msg := sprintf("%s: pod securityContext.runAsNonRoot має бути true", [input.metadata.name])
}

deny contains msg if {
	some c in containers
	not c.securityContext.allowPrivilegeEscalation == false
	msg := sprintf("%s/%s: allowPrivilegeEscalation має бути false", [input.metadata.name, c.name])
}

warn contains msg if {
	some c in containers
	not c.securityContext.readOnlyRootFilesystem
	msg := sprintf("%s/%s: рекомендовано readOnlyRootFilesystem", [input.metadata.name, c.name])
}
