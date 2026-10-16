# Golden path: кожен компонент зареєстрований у каталозі з власником, життєвим циклом і системою.
package catalog

import rego.v1

deny contains msg if {
	input.kind == "Component"
	some field in ["owner", "lifecycle", "type", "system"]
	not input.spec[field]
	msg := sprintf("Component %s: відсутнє spec.%s", [input.metadata.name, field])
}

deny contains msg if {
	input.kind == "Component"
	not input.metadata.description
	msg := sprintf("Component %s: відсутній опис", [input.metadata.name])
}
