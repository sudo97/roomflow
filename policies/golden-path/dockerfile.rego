# Golden path: вимоги до Dockerfile сервісів платформи (conftest, парсер dockerfile).
package dockerfile

import rego.v1

stages := [i | some i, cmd in input; cmd.Cmd == "from"]

# Імена етапів (FROM image AS name): посилання на попередній етап — не базовий образ.
stage_names := {lower(cmd.Value[2]) | some cmd in input; cmd.Cmd == "from"; count(cmd.Value) == 3}

external(image) if not lower(image) in stage_names

deny contains msg if {
	some cmd in input
	cmd.Cmd == "from"
	image := cmd.Value[0]
	external(image)
	not contains(image, ":")
	msg := sprintf("FROM %s: версію базового образу треба зафіксувати", [image])
}

deny contains msg if {
	some cmd in input
	cmd.Cmd == "from"
	endswith(cmd.Value[0], ":latest")
	msg := sprintf("FROM %s: тег latest заборонено", [cmd.Value[0]])
}

# Останній етап має перемикатися на непривілейованого користувача.
deny contains msg if {
	last_from := stages[count(stages) - 1]
	users := [u | some i, cmd in input; i > last_from; cmd.Cmd == "user"; u := cmd.Value[0]]
	not non_root(users)
	msg := "фінальний етап образу має виконуватися від непривілейованого користувача (USER)"
}

non_root(users) if {
	count(users) > 0
	u := users[count(users) - 1]
	not u in {"root", "0", "0:0", "root:root"}
}
