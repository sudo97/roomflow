# Політики безпеки та витрат для планів Terraform (Open Policy Agent / conftest).
# Вхідні дані: terraform show -json plan.out. Порушення блокують застосування змін у конвеєрі.
package main

import rego.v1

required_labels := {"project", "environment", "owner", "cost_center"}

sensitive_components := {"database", "database-backup"}

max_memory_mb := 512

changes contains rc if {
	some rc in input.resource_changes
	not "delete" in rc.change.actions
	rc.change.after != null
}

containers contains rc if {
	some rc in changes
	rc.type == "docker_container"
}

label_map(after) := {l.label: l.value | some l in after.labels}

# 1. Обов'язкове тегування: облік витрат за проєктом, середовищем, власником і центром витрат.
deny contains msg if {
	some rc in changes
	rc.type in {"docker_container", "docker_network", "docker_volume"}
	missing := required_labels - object.keys(label_map(rc.change.after))
	count(missing) > 0
	msg := sprintf("%s: відсутні обов'язкові мітки %v", [rc.address, missing])
}

# 2. Чутливі сервіси (база даних) не можуть мати публічного доступу.
deny contains msg if {
	some rc in containers
	label_map(rc.change.after).component in sensitive_components
	count(object.get(rc.change.after, "ports", [])) > 0
	msg := sprintf("%s: база даних не повинна публікувати порти назовні", [rc.address])
}

# 3. Заборона привілейованих контейнерів.
deny contains msg if {
	some rc in containers
	rc.change.after.privileged == true
	msg := sprintf("%s: привілейовані контейнери заборонені", [rc.address])
}

# 4. Контроль витрат: кожен контейнер має ліміт пам'яті не більше max_memory_mb (аналог дозволених типів інстансів).
deny contains msg if {
	some rc in containers
	memory := object.get(rc.change.after, "memory", 0)
	not memory_ok(memory)
	msg := sprintf("%s: ліміт пам'яті %v МБ поза межами (0, %v]", [rc.address, memory, max_memory_mb])
}

memory_ok(m) if {
	m > 0
	m <= max_memory_mb
}

# 5. Відтворюваність: образи з конкретною версією, тег latest або його відсутність заборонені.
deny contains msg if {
	some rc in changes
	rc.type == "docker_image"
	name := rc.change.after.name
	not pinned(name)
	msg := sprintf("%s: образ %s має бути із зафіксованою версією", [rc.address, name])
}

pinned(name) if {
	parts := split(name, "/")
	last := parts[count(parts) - 1]
	contains(last, ":")
	not endswith(last, ":latest")
}

# Попередження (не блокують): мережі без підмережі у явному вигляді.
warn contains msg if {
	some rc in changes
	rc.type == "docker_network"
	count(object.get(rc.change.after, "ipam_config", [])) == 0
	msg := sprintf("%s: підмережу не задано явно", [rc.address])
}
