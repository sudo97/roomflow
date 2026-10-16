package main

import rego.v1

container(address, after) := {"address": address, "type": "docker_container", "change": {"actions": ["create"], "after": after}}

labels(m) := [{"label": k, "value": v} | some k, v in m]

good_labels := labels({"project": "roomflow", "environment": "ci", "owner": "team", "cost_center": "edu", "component": "app"})

test_compliant_container_passes if {
	count(deny) == 0 with input as {"resource_changes": [container("a", {"labels": good_labels, "memory": 128, "ports": []})]}
}

test_missing_labels_denied if {
	some msg in deny with input as {"resource_changes": [container("a", {"labels": labels({"project": "roomflow"}), "memory": 128})]}
	contains(msg, "обов'язкові мітки")
}

test_public_database_denied if {
	db_labels := labels({"project": "roomflow", "environment": "ci", "owner": "team", "cost_center": "edu", "component": "database"})
	some msg in deny with input as {"resource_changes": [container("db", {"labels": db_labels, "memory": 128, "ports": [{"internal": 5432, "external": 5432}]})]}
	contains(msg, "база даних")
}

test_privileged_denied if {
	some msg in deny with input as {"resource_changes": [container("p", {"labels": good_labels, "memory": 128, "privileged": true})]}
	contains(msg, "привілейовані")
}

test_memory_limit_required if {
	some msg in deny with input as {"resource_changes": [container("m", {"labels": good_labels})]}
	contains(msg, "ліміт пам'яті")
}

test_memory_over_budget_denied if {
	some msg in deny with input as {"resource_changes": [container("m", {"labels": good_labels, "memory": 4096})]}
	contains(msg, "ліміт пам'яті")
}

test_latest_image_denied if {
	img := {"address": "i", "type": "docker_image", "change": {"actions": ["create"], "after": {"name": "nginx:latest"}}}
	some msg in deny with input as {"resource_changes": [img]}
	contains(msg, "зафіксованою версією")
}

test_untagged_registry_image_denied if {
	img := {"address": "i", "type": "docker_image", "change": {"actions": ["create"], "after": {"name": "localhost:5000/roomflow"}}}
	count(deny) == 1 with input as {"resource_changes": [img]}
}

test_pinned_registry_image_allowed if {
	img := {"address": "i", "type": "docker_image", "change": {"actions": ["create"], "after": {"name": "localhost:5000/roomflow:v1"}}}
	count(deny) == 0 with input as {"resource_changes": [img]}
}

test_deletes_are_ignored if {
	rc := {"address": "x", "type": "docker_container", "change": {"actions": ["delete"], "after": null}}
	count(deny) == 0 with input as {"resource_changes": [rc]}
}
